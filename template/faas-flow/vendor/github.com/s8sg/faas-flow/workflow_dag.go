package faasflow

import (
	sdk "github.com/s8sg/faas-flow/sdk"
)

// CreateDag creates a new dag definition
func CreateDag() *DagFlow {
	dag := &DagFlow{}
	udag := sdk.NewDag()
	dag.udag = udag

	return dag
}

// AppndDag appends a seperate dag into current dag
// provided dag should be mutually exclusive
func (this *DagFlow) AppndDag(dag *DagFlow) error {
	return this.udag.Append(dag.udag)
}

// AppndDagEdge appends a seperate dag between two given vertex
// provided dag should be mutually exclusive
func (this *DagFlow) AppndDagEdge(from, to string, dag *DagFlow) error {
	err := this.udag.Append(dag.udag)
	if err != nil {
		return err
	}
	dagInitialNode := dag.udag.GetInitialNode().Id
	dagEndNode := dag.udag.GetEndNode().Id
	err = this.AddEdge(from, dagInitialNode)
	if err != nil {
		return err
	}
	err = this.AddEdge(dagEndNode, to)
	if err != nil {
		return err
	}
	return nil
}

// AddVertex add a new vertex by id
// If exist overrides the vertex settings
func (this *DagFlow) AddVertex(vertex string, opts ...Option) {
	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if o.serializer != nil {
			// Add serializer only on node declearation
			node.AddSerializer(o.serializer)
		}
	}
}

// AddDag adds a seperate dag to the given vertex
// if vertex already exist it will replace the existing definition
// if not new vertex will be created
func (this *DagFlow) AddDag(vertex string, dag *DagFlow) error {

	node := this.udag.GetNode(vertex)

	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	return node.AddSubDag(dag.udag)
}

// AddModifier adds a new modifier to the given vertex
// if vertex already exist it will be appended at the end,
// if not new vertex will be created
func (this *DagFlow) AddModifier(vertex string, mod sdk.Modifier) {
	newMod := sdk.CreateModifier(mod)

	node := this.udag.GetNode(vertex)

	if node == nil {
		this.udag.AddVertex(vertex, []*sdk.Operation{newMod})
	} else {
		node.AddOperation(newMod)
	}
}

// AddFunction adds a new function to the given vertex
// if vertex already exist it will be appended at the end,
// if not new vertex will be created
func (this *DagFlow) AddFunction(vertex string, function string, opts ...Option) {
	newfunc := sdk.CreateFunction(function)

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newfunc.Addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newfunc.Addparam(key, value)
				}
			}
		}
		if o.failureHandler != nil {
			newfunc.AddFailureHandler(o.failureHandler)
		}
		if o.responseHandler != nil {
			newfunc.AddResponseHandler(o.responseHandler)
		}
	}

	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{newfunc})
	} else {
		node.AddOperation(newfunc)
	}
}

// AddCallback adds a new callback to the given vertex
// if vertex already exist it will be appended at the end,
// if not new vertex will be created
func (this *DagFlow) AddCallback(vertex string, url string, opts ...Option) {
	newCallback := sdk.CreateCallback(url)

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newCallback.Addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newCallback.Addparam(key, value)
				}
			}
		}
		if o.failureHandler != nil {
			newCallback.AddFailureHandler(o.failureHandler)
		}
	}

	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{newCallback})
	} else {
		node.AddOperation(newCallback)
	}
}

// AddEdge adds a directed edge between two vertex as <from>-><to>
func (this *DagFlow) AddEdge(from, to string, opts ...Option) error {
	err := this.udag.AddEdge(from, to)
	if err != nil {
		return err
	}

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if o.noforwarder == true {
			fromNode := this.udag.GetNode(from)
			// Add a nil forwarder
			fromNode.AddForwarder(to, nil)
		}
		if o.forwarder != nil {
			fromNode := this.udag.GetNode(from)
			// Add serializer only on node declearation
			fromNode.AddForwarder(to, o.forwarder)
		}
	}

	return nil
}
