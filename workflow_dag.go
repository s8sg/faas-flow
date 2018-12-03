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

// CreateModifierVertex create a new modifier that can be added in dag
// if vertex already exist it will be appended at the end
func (this *DagFlow) CreateModifierVertex(id string, mod sdk.Modifier, opts ...Option) {
	newMod := sdk.CreateModifier(mod)

	node := this.udag.GetNode(id)
	o := &Options{}

	if node == nil {
		node = this.udag.AddVertex(id, []*sdk.Operation{newMod})
		for _, opt := range opts {
			o.reset()
			opt(o)
			if o.serializer != nil {
				// Add serializer only on node declearation
				node.AddSerializer(o.serializer)
			}
		}
	} else {
		node.AddOperation(newMod)
	}
}

// CreateFunctionVertex create a new function that can be added in the dag
// if vertex already exist it will be appended at the end
func (this *DagFlow) CreateFunctionVertex(id string, function string, opts ...Option) {
	newfunc := sdk.CreateFunction(function)

	node := this.udag.GetNode(id)
	o := &Options{}

	if node == nil {
		node = this.udag.AddVertex(id, []*sdk.Operation{newfunc})
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

			// Add serializer only on node declearation
			if o.serializer != nil {
				node.AddSerializer(o.serializer)
			}
		}
	} else {
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
		node.AddOperation(newfunc)
	}
}

// CreateCallbackVertex create a new callback that can be added as a dag
// if vertex already exist it will be appended at the end
func (this *DagFlow) CreateCallbackVertex(id string, url string, opts ...Option) {
	newCallback := sdk.CreateCallback(url)

	node := this.udag.GetNode(id)
	o := &Options{}

	if node == nil {
		node = this.udag.AddVertex(id, []*sdk.Operation{newCallback})
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
			if o.serializer != nil {
				node.AddSerializer(o.serializer)
			}
		}
	} else {
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
		node.AddOperation(newCallback)
	}
}

// AddEdge adds a directed edge as <from>-><to>
func (this *DagFlow) AddEdge(from, to string) error {
	return this.udag.AddEdge(from, to)
}
