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

// AddVertex add a new vertex by id
// If exist overrides the vertex settings
func (this *DagFlow) AddVertex(id string, opts ...Option) {
	node := this.udag.GetNode(id)
	if node == nil {
		node = this.udag.AddVertex(id, []*sdk.Operation{})
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

// CreateModifierVertex create a new modifier that can be added in dag
// if vertex already exist it will be appended at the end,
// if not new vertex will be created
func (this *DagFlow) AddModifier(id string, mod sdk.Modifier) {
	newMod := sdk.CreateModifier(mod)

	node := this.udag.GetNode(id)

	if node == nil {
		this.udag.AddVertex(id, []*sdk.Operation{newMod})
	} else {
		node.AddOperation(newMod)
	}
}

// CreateFunctionVertex create a new function that can be added in the dag
// if vertex already exist it will be appended at the end,
// if not new vertex will be created
func (this *DagFlow) AddFunction(id string, function string, opts ...Option) {
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

	node := this.udag.GetNode(id)
	if node == nil {
		node = this.udag.AddVertex(id, []*sdk.Operation{newfunc})
	} else {
		node.AddOperation(newfunc)
	}
}

// CreateCallbackVertex create a new callback that can be added as a dag
// if vertex already exist it will be appended at the end,
// if not new vertex will be created
func (this *DagFlow) AddCallback(id string, url string, opts ...Option) {
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

	node := this.udag.GetNode(id)
	if node == nil {
		node = this.udag.AddVertex(id, []*sdk.Operation{newCallback})
	} else {
		node.AddOperation(newCallback)
	}
}

// AddEdge adds a directed edge as <from>-><to>
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
