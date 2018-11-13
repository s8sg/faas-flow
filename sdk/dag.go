package sdk

import (
	"fmt"
	//       "github.com/funkygao/golib/str"
)

var (
	ErrCyclic = fmt.Errorf("dag has cyclic dependency")
)

type Dag struct {
	nodes  map[string]*Node
	status map[string]bool
}

type Node struct {
	id  string
	val *Function

	outdegree int
	indegree  int

	children []*Node
	//childrenSync map[string]bool

	next []*Node
	prev []*Node
}

func NewDag() *Dag {
	this := new(Dag)
	this.nodes = make(map[string]*Node)
	return this
}

func (this *Dag) AddVertex(id string, val *Function) *Node {
	node := &Node{id: id, val: val}
	this.nodes[id] = node
	this.status[id] = false
	//node.childrenSync = new(map[string]bool)
	return node
}

func (this *Node) inSlice(list []*Node) bool {
	for _, b := range list {
		if b.id == this.id {
			return true
		}
	}
	return false
}

func (this *Dag) AddEdge(from, to string) error {
	fromNode := this.nodes[from]
	toNode := this.nodes[to]

	// Check if cyclic dependency
	if fromNode.inSlice(toNode.next) || toNode.inSlice(fromNode.prev) {
		return ErrCyclic
	}

	fromNode.next = append(fromNode.next, toNode)
	fromNode.next = append(fromNode.next, toNode.next...)
	for _, b := range fromNode.prev {
		b.next = append(b.next, toNode)
		b.next = append(b.next, toNode.next...)
	}

	toNode.prev = append(toNode.prev, fromNode)
	toNode.prev = append(toNode.prev, fromNode.prev...)
	for _, b := range toNode.next {
		b.prev = append(b.prev, fromNode)
		b.prev = append(b.prev, fromNode.prev...)
	}

	fromNode.children = append(fromNode.children, toNode)
	//fromNode.childrenSync = sync

	fromNode.outdegree++
	toNode.indegree++

	return nil
}

func (this *Dag) Node(id string) *Node {
	return this.nodes[id]
}

/*
func (this *Dag) MakeDotGraph(fileName string) string {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	sb := str.NewStringBuilder()
	sb.WriteString("digraph depgraph {\n\trankdir=LR;\n")
	for _, node := range this.nodes {
		node.dotGraph(sb)
	}
	sb.WriteString("}\n")
	file.WriteString(sb.String())
	return sb.String()
}

func (this *Node) dotGraph(sb *str.StringBuilder) {
	if len(this.children) == 0 {
		sb.WriteString(fmt.Sprintf("\t\"%s\";\n", this.id))
		return
	}

	for _, child := range this.children {
		sb.WriteString(fmt.Sprintf(`%s -> %s [label="%v"]`, this.id, child.id, this.val))
		sb.WriteString("\r\n")
	}
}*/

func (this *Node) Children() []*Node {
	return this.children
}
