package sdk

import (
	"fmt"
)

var (
	// ERR_CYCLIC denotes that dag has a cycle
	ERR_CYCLIC = fmt.Errorf("dag has cyclic dependency")
	// ERR_DUPLICATE_EDGE denotes that a dag edge is duplicate
	ERR_DUPLICATE_EDGE = fmt.Errorf("edge redefined")
	// ERR_DUPLICATE_VERTEX denotes that a dag edge is duplicate
	ERR_DUPLICATE_VERTEX = fmt.Errorf("vertex redefined")
	// ERR_MULTIPLE_START denotes that a dag has more than one start point
	ERR_MULTIPLE_START = fmt.Errorf("only one start vertex is allowed")
	// ERR_MULTIPLE_END denotes that a dag has more than one end point
	ERR_MULTIPLE_END = fmt.Errorf("only one end vertex is allowed")
	// ERR_RECURSIVE_DEP denotes that dag has a recursive dependecy
	ERR_RECURSIVE_DEP = fmt.Errorf("dag has recursive dependency")
	// NodeIndex
	nodeIndex = 0
	// Default forwarder
	DefaultForwarder = func(data []byte) []byte { return data }
)

// Serializer defintion for the data serilizer of nodes
type Serializer func(map[string][]byte) ([]byte, error)

// Forwarder defintion for the data forwarder of nodes
type Forwarder func([]byte) []byte

// Dag The whole dag
type Dag struct {
	Id    string
	nodes map[string]*Node // the nodes in a dag

	parentNode *Node // In case the dag is a sub dag the node reference

	initialNode *Node // The start of a valid dag
	endNode     *Node // The end of a valid dag
}

// Node The vertex
type Node struct {
	Id    string // The id of the vertex
	index int    // The index of the vertex

	// Execution modes ([]operation / Dag)
	subDag     *Dag         // Subdag
	operations []*Operation // The list of operations

	serializer Serializer           // The serializer serialize multiple input to a node into one
	forwarder  map[string]Forwarder // The forwarder handle forwarding output to a children

	parentDag *Dag    // The reference of the dag this node part of
	indegree  int     // The vertex dag indegree
	outdegree int     // The vertex dag outdegree
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

// Append appends another dag into an existing dag
// Its a way to define and reuse subdags
// append causes disconnected dag which must be linked with edge in order to execute
func (this *Dag) Append(dag *Dag) error {
	err := dag.Validate()
	if err != nil {
		return err
	}
	for nodeId, node := range dag.nodes {
		_, duplicate := this.nodes[nodeId]
		if duplicate {
			return ERR_DUPLICATE_VERTEX
		}
		// add the node
		this.nodes[nodeId] = node
	}
	return nil
}

// AddVertex create a vertex with id and operations
func (this *Dag) AddVertex(id string, operations []*Operation) *Node {

	node := &Node{Id: id, operations: operations, index: nodeIndex + 1}
	node.forwarder = make(map[string]Forwarder, 0)
	node.parentDag = this
	nodeIndex = nodeIndex + 1
	this.nodes[id] = node
	return node
}

// AddEdge add a directed edge as (from)->(to)
// If vertex doesn't exists creates them
func (this *Dag) AddEdge(from, to string) error {
	fromNode := this.nodes[from]
	if fromNode == nil {
		fromNode = this.AddVertex(from, []*Operation{})
	}
	toNode := this.nodes[to]
	if toNode == nil {
		toNode = this.AddVertex(to, []*Operation{})
	}

	// CHeck if duplicate (TODO: Check if one way check is enough)
	if toNode.inSlice(fromNode.children) || fromNode.inSlice(toNode.dependsOn) {
		return ERR_DUPLICATE_EDGE
	}

	// Check if cyclic dependency (TODO: Check if one way check if enough)
	if fromNode.inSlice(toNode.next) || toNode.inSlice(fromNode.prev) {
		return ERR_CYCLIC
	}

	// Update references recursively
	fromNode.next = append(fromNode.next, toNode)
	fromNode.next = append(fromNode.next, toNode.next...)
	for _, b := range fromNode.prev {
		b.next = append(b.next, toNode)
		b.next = append(b.next, toNode.next...)
	}

	// Update references recursively
	toNode.prev = append(toNode.prev, fromNode)
	toNode.prev = append(toNode.prev, fromNode.prev...)
	for _, b := range toNode.next {
		b.prev = append(b.prev, fromNode)
		b.prev = append(b.prev, fromNode.prev...)
	}

	fromNode.children = append(fromNode.children, toNode)
	toNode.dependsOn = append(toNode.dependsOn, fromNode)
	toNode.indegree++
	fromNode.outdegree++

	// Add default forwarder for from node
	fromNode.AddForwarder(to, DefaultForwarder)

	return nil
}

// GetNode get a node by Id
func (this *Dag) GetNode(id string) *Node {
	return this.nodes[id]
}

// GetParentNode returns parent node for a subdag
func (this *Dag) GetParentNode() *Node {
	return this.parentNode
}

// GetInitialNode gets the initial node
func (this *Dag) GetInitialNode() *Node {
	return this.initialNode
}

// GetEndNode gets the end node
func (this *Dag) GetEndNode() *Node {
	return this.endNode
}

// Validate validates a dag as per faas-flow dag requirments
func (this *Dag) Validate() error {
	initialNodeCount := 0
	endNodeCount := 0

	for _, b := range this.nodes {
		if b.indegree == 0 {
			initialNodeCount = initialNodeCount + 1
			this.initialNode = b
		}
		if b.outdegree == 0 {
			endNodeCount = endNodeCount + 1
			this.endNode = b
		}
		if b.subDag != nil {
			err := b.subDag.Validate()
			if err != nil {
				return err
			}
		}
	}

	if initialNodeCount > 1 {
		return ERR_MULTIPLE_START
	}
	if endNodeCount > 1 {
		return ERR_MULTIPLE_END
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

// Outdegree returns the no of output in a node
func (this *Node) Outdegree() int {
	return this.outdegree
}

// SubDag returns the subdag added in a node
func (this *Node) SubDag() *Dag {
	return this.subDag
}

// ParentDag returns the parent dag of the node
func (this *Node) ParentDag() *Dag {
	return this.parentDag
}

// AddOperation adds an operation
func (this *Node) AddOperation(operation *Operation) {
	this.operations = append(this.operations, operation)
}

// AddSerializer add a serializer to a node
func (this *Node) AddSerializer(serializer Serializer) {
	this.serializer = serializer
}

// AddForwarder adds a forwarder for a specific children
func (this *Node) AddForwarder(children string, forwarder Forwarder) {
	this.forwarder[children] = forwarder
}

// AddSubDag adds a subdag to the node
func (this *Node) AddSubDag(subDag *Dag) error {
	parent := this.parentDag
	for parent != nil {
		if parent == subDag {
			return ERR_RECURSIVE_DEP
		}
		parentNode := parent.parentNode
		if parentNode != nil {
			parent = parentNode.subDag
			continue
		}
		break
	}
	this.subDag = subDag
	// Set the node the subdag belongs to
	subDag.parentNode = this
	subDag.Id = this.Id

	return nil
}

// GetSerializer get a serializer from a node
func (this *Node) GetSerializer() Serializer {
	return this.serializer
}

// GetForwarder gets a forwarder for a children
func (this *Node) GetForwarder(children string) Forwarder {
	return this.forwarder[children]
}
