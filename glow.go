package glow

import (
	"fmt"
	"reflect"
	"strings"
)

type Argument struct {
	name string
	val  reflect.Value
	Type reflect.Type
}

// ----------------------------------------------------------------------------
type Node struct {
	name string        // The node's name.
	fi   interface{}   // The node's run function.
	ft   reflect.Type  // The function's type.
	fv   reflect.Value // The function's value.
	args []Argument
}

func NewNode(fn interface{}, name string, argNames ...string) *Node {
	node := &Node{
		name: name,
		fi:   fn,
		ft:   reflect.TypeOf(fn),
		fv:   reflect.ValueOf(fn),
	}

	argNames = append([]string{"globals"}, argNames...)

	for i, argName := range argNames {
		var arg Argument
		if node.ft.IsVariadic() && i >= node.ft.NumIn()-1 {
			lastArg := node.ft.In(node.ft.NumIn() - 1)
			arg = Argument{name: argName, Type: lastArg.Elem()}
		} else {
			arg = Argument{name: argName, Type: node.ft.In(i)}
		}
		node.args = append(node.args, arg)
	}
	return node
}

func (node *Node) Run() {
	values := make([]reflect.Value, len(node.args))
	for i, arg := range node.args {
		values[i] = arg.val
	}
	node.fv.Call(values)
}

func (node *Node) MakeChan(name string, size int) reflect.Value {
	for _, arg := range node.args {
		if arg.name == name {
			return reflect.MakeChan(arg.Type, size)
		}
	}
	panic("Argument not found.")
}

func (node *Node) SetArg(name string, val reflect.Value) {
	for i, arg := range node.args {
		if arg.name == name {
			if arg.val.IsValid() {
				panic("Argument alread set: " + name)
			}
			node.args[i].val = val
			return
		}
	}
	panic("Argument not found.")
}

func (node *Node) DotString() string {
	s := node.name + " [\n"
	s += "label = \"" + node.name
	for _, arg := range node.args[1:] {
		s += "|<" + arg.name + ">" + arg.name
	}
	s += "\"\n"
	s += "shape = record\n]"
	return s
}

// ----------------------------------------------------------------------------
type Graph struct {
	connStr  []string         // List of connections for dot file output.
	lastChan int              // Last channel number for dot file output.
	nodes    map[string]*Node // Map from node name to node.
	globals  reflect.Value    // Globals passed to each node.
	fgName   string           // Name of node to run in the foreground.
}

// NewGraph: Construct a new empty graph object. The value of globals
// will be passed as the first argument to each node function.
func NewGraph(globals interface{}) *Graph {
	graph := new(Graph)
	graph.globals = reflect.ValueOf(globals)
	graph.nodes = make(map[string]*Node)
	return graph
}

// AddNode: Add a new node to the graph. A node is implemented by a function,
// fn, and has a unique identifying name. Names of function arguments after
// the first must be given. The first argument will be the value of
// globals given when creating the graph.
func (g *Graph) AddNode(fn interface{}, name string, argNames ...string) {
	// If the node name is already in use, this is a programming error.
	_, ok := g.nodes[name]
	if ok {
		panic("Node already added: " + name)
	}
	node := NewNode(fn, name, argNames...)
	node.SetArg("globals", g.globals)
	g.nodes[name] = node
}

// Connect: Create a channel of the appropriate type to be passed to the
// given node's implementing function when the graph is run. The size of
// the channel buffer is the first argument. Additional arguments list the
// nodes that will be using the channel. The format for these arguments is
// "NodeName:ChannelName".
// Returns the new channel as a reflect.Value.
func (g *Graph) Connect(size int, nodeChans ...string) reflect.Value {
	name, port := splitNamePort(nodeChans[0])
	ch := g.nodes[name].MakeChan(port, size)

	chName := fmt.Sprintf("chan_%v", g.lastChan)
	g.lastChan += 1
	g.connStr = append(g.connStr,
		fmt.Sprintf("%v [\nlabel=\"%v\"\n]", chName, size))

	for _, s := range nodeChans {
		name, port = splitNamePort(s)
		g.nodes[name].SetArg(port, ch)

		if strings.HasSuffix(port, "Out") {
			g.connStr = append(g.connStr, name+":"+port+"->"+chName)
		} else {
			g.connStr = append(g.connStr, chName+"->"+name+":"+port)
		}
	}
	return ch
}

// SetForeground: Specify a node to run in the foreground when Run is called
// on the graph.
func (g *Graph) SetForeground(name string) {
	g.fgName = name
}

// DotString: Return a string containing a dot file suitable for processing
// by graphviz. On Linux, xdot can be used to view a dot file directly.
func (g *Graph) DotString() string {
	s := "digraph {"
	s += "\ngraph [ rankdir=\"LR\" ];"
	for _, node := range g.nodes {
		s += "\n" + node.DotString()
	}
	for _, conn := range g.connStr {
		s += "\n" + conn
	}
	s += "\n}"
	return s
}

// Run: Run each of the graph's nodes in a goroutine, with the exception of an
// optionally defined foreground node, which will run in the foreground.
func (g *Graph) Run() {
	var fgNode *Node

	for _, node := range g.nodes {
		if node.name != g.fgName {
			go node.Run()
		} else {
			fgNode = node
		}
	}

	if fgNode != nil {
		fgNode.Run()
	}
}

// ----------------------------------------------------------------------------
func splitNamePort(s string) (string, string) {
	x := strings.Split(s, ":")
	return x[0], x[1]
}
