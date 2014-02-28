glow
====

glow is a simple package that provides a framework for dataflow-like 
programming in go. Nodes are implemented as go functions, and are connected 
by channels. 

Nodes
-----

Nodes are implemented as by functions. The first argument to all nodes in a 
graph is the graph's globals. Additional arguments must be channels for input 
and output. By convention these channels have either "In" or "Out" as a suffix.

```go

func ExampleNode(gl *Globals, StrIn, StrOut chan string) {
     // Some code here....
}
```

Graphs
------

A graph consists of a series of nodes that are connected by channels. 

```go

g := glow.NewGraph(&globals)

// Add nodes.
g.AddNode(ExampleNode, "Node1", "StrIn", "StrOut")
g.AddNode(SomeOtherNode, "Node2", "StrIn")

// Make connections. 
g.Connect(1, "Node1:StrOut", "Node2:StrIn")

// Print dot file string.
fmt.Println(g.DotString())

// Set a foreground node. 
g.SetForeground("Node2")

// Run. 
g.Run()
```