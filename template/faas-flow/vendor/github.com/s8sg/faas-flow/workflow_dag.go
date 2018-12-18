package faasflow

import (
	"fmt"
	sdk "github.com/s8sg/faas-flow/sdk"
)

var (
	INVAL_OPTION = fmt.Errorf("invalid option specified")
)

// CreateDag creates a new dag definition
func CreateDag() *DagFlow {
	dag := &DagFlow{}
	udag := sdk.NewDag()
	dag.udag = udag

	return dag
}

// AppendDag generalizes a seperate dag by appending its properties into current dag.
// Provided dag should be mutually exclusive
func (this *DagFlow) AppendDag(dag *DagFlow) error {
	return this.udag.Append(dag.udag)
}

// AddVertex add a new vertex by id
// If exist overrides the vertex settings.
// Allowed option: Aggregator, ForEach
func (this *DagFlow) AddVertex(vertex string, opts ...Option) {
	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if o.aggregator != nil {
			// Add aggregator only on node declearation
			node.AddAggregator(o.aggregator)
		}
	}
}

// AddSubDag composites a seperate dag as a subdag to the given vertex.
// If vertex already exist it will override the existing definition,
// If not new vertex will be created.
// When a vertex is dag, operations are ommited.
func (this *DagFlow) AddSubDag(vertex string, dag *DagFlow) error {

	node := this.udag.GetNode(vertex)

	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	return node.AddSubDag(dag.udag)
}

// AddForEachDag composites a seperate dag as a subdag which executes for each value
// If vertex already exist it will override the existing definition,
// If not new vertex will be created.
// When a vertex is dag, operations are ommited.
// foreach: ForEach Option
func (this *DagFlow) AddForEachDag(vertex string, dag *DagFlow, foreach Option) error {
	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	o := &Options{}
	foreach(o)
	if o.foreach != nil {
		node.AddForEach(o.foreach)
	} else {
		return INVAL_OPTION
	}
	if o.aggregator != nil {
		node.AddSubAggregator(o.aggregator)
	} else {
		return INVAL_OPTION
	}
	return node.AddSubDag(dag.udag)
}

// AddConditionalDag composites multiple seperate dag as a subdag which executes for a conditions matched
// If vertex already exist it will override the existing definition,
// If not new vertex will be created.
// When a vertex is dag, operations are ommited.
// condition: Condition Option
func (this *DagFlow) AddConditionalDags(vertex string, subdags map[string]*DagFlow, condition Option) error {
	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	o := &Options{}
	condition(o)
	if o.condition != nil {
		node.AddCondition(o.condition)
	} else {
		return INVAL_OPTION
	}
	if o.aggregator != nil {
		node.AddSubAggregator(o.aggregator)
	} else {
		return INVAL_OPTION
	}
	for conditionKey, dag := range subdags {
		err := node.AddConditionalDag(conditionKey, dag.udag)
		if err != nil {
			return err
		}
	}
	return nil
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
			// Add aggregator only on node declearation
			fromNode.AddForwarder(to, o.forwarder)
		}
	}

	return nil
}
