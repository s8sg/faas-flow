package faasflow

import (
	sdk "github.com/s8sg/faas-flow/sdk"
)

func CreateDag() *DagFlow {
	dag := &DagFlow{}
	udag := sdk.NewDag()
	dag.udag = udag

	return dag
}

// ModifierVertex create a new modifier that can be added in dag
func (this *DagFlow) ModifierVertex(id string, mod sdk.Modifier, opts ...Option) {
	newMod := sdk.CreateModifier(mod)
	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if o.serializer != nil {
			newMod.AddSerializer(o.serializer)
		}
	}
	this.udag.AddVertex(id, newMod)
}

// FunctionVertex create a new function that can be added in the dag
func (this *DagFlow) FunctionVertex(id string, function string, opts ...Option) {
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

		if o.serializer != nil {
			newfunc.AddSerializer(o.serializer)
		}
	}
	this.udag.AddVertex(id, newfunc)
}

// CallbackVertex create a new callback that can be added as a dag
func (this *DagFlow) CallbackVertex(id string, url string, opts ...Option) {
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
		if o.serializer != nil {
			newCallback.AddSerializer(o.serializer)
		}
	}
	this.udag.AddVertex(id, newCallback)
}

// AddEdge adds a directed edge as <from>-><to>
func (this *DagFlow) AddEdge(from, to string) error {
	return this.udag.AddEdge(from, to)
}
