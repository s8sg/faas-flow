package sdk

import (
	"fmt"
)

var (
	// ERR_CYCLIC denotes that dag has a cycle
	ERR_CYCLIC = fmt.Errorf("dag has cyclic dependency")
)

// Serializer defintion for the data serilizer of function
type Serializer func(map[string][]byte) ([]byte, error)

// Dag The whole dag
type Dag struct {
	nodes map[string]*Node
}

// Node The vertex
type Node struct {
	Id string // The id of the vertex

	operations []*Operation // The list of operations
	serializer Serializer   // The serializer serialize multiple input to a node into one

	indegree  int     // The vertex dag indegree
	children  []*Node // The children of the vertex
	dependsOn []*Node // The parents of the vertex

	next []*Node
	prev []*Node
}

// NewDag Creates a Dag
func NewDag() *Dag {
	this := new(Dag)
	this.nodes = make(map[string]*Node)
	return this
}

// AddVertex create a vertex with id and operations
func (this *Dag) AddVertex(id string, operations []*Operation) *Node {
	node := &Node{Id: id, operations: operations}
	this.nodes[id] = node
	return node
}

// AddEdge add a directed edge as (from)->(to)
func (this *Dag) AddEdge(from, to string) error {
	fromNode := this.nodes[from]
	toNode := this.nodes[to]

	// Check if cyclic dependency
	if fromNode.inSlice(toNode.next) || toNode.inSlice(fromNode.prev) {
		return ERR_CYCLIC
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
	toNode.dependsOn = append(toNode.dependsOn, fromNode)
	toNode.indegree++

	return nil
}

// GetNode get a node by Id
func (this *Dag) GetNode(id string) *Node {
	return this.nodes[id]
}

// GetInitialNode get the initial node
func (this *Dag) GetInitialNode() *Node {
	for _, b := range this.nodes {
		if b.indegree == 0 {
			return b
		}
	}
	return nil
}

// inSlice check if a node belongs in a slice
func (this *Node) inSlice(list []*Node) bool {
	for _, b := range list {
		if b.Id == this.Id {
			return true
		}
	}
	return false
}

// Children get all children node for a node
func (this *Node) Children() []*Node {
	return this.children
}

// Dependency get all dependency node for a node
func (this *Node) Dependency() []*Node {
	return this.dependsOn
}

// Value provides the ordered list of functions for a node
func (this *Node) Operations() []*Operation {
	return this.operations
}

// Indegree returns the no of input in a node
func (this *Node) Indegree() int {
	return this.indegree
}

// AddOperation add an operation
func (this *Node) AddOperation(operation *Operation) {
	this.operations = append(this.operations, operation)
}

// AddSerializer add a serializer to a node
func (this *Node) AddSerializer(serializer Serializer) {
	this.serializer = serializer
}

// GetSerializer get a serializer from a node
func (this *Node) GetSerializer() Serializer {
	return this.serializer
}
