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
	dag.udag = sdk.NewDag()

	return dag
}

// AppendDag generalizes a seperate dag by appending its properties into current dag.
// Provided dag should be mutually exclusive
func (this *DagFlow) AppendDag(dag *DagFlow) {
	err := this.udag.Append(dag.udag)
	if err != nil {
		panic(fmt.Sprintf("Error at AppendDag, %v", err))
	}
}

// AddVertex adds a new vertex by id
// If exist overrides the vertex settings.
// options: Aggregator
func (this *DagFlow) AddVertex(vertex string, options ...BranchOption) {
	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	o := &BranchOptions{}
	for _, opt := range options {
		o.reset()
		opt(o)
		if o.aggregator != nil {
			node.AddAggregator(o.aggregator)
		}
	}
}

// AddSubDag composites a seperate dag as a subdag to the given vertex.
// If vertex already exist it will override the existing definition,
// If not new vertex will be created.
// When a vertex is dag, operations are ommited.
/*
func (this *DagFlow) AddSubDag(vertex string, dag *DagFlow) {

	node := this.udag.GetNode(vertex)

	if node == nil {
		node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	}
	err := node.AddSubDag(dag.udag)
	if err != nil {
		panic(fmt.Sprintf("Error at AddSubDag for %s, %v", vertex, err))
	}
}*/

// AddForEachBranch composites a subdag which executes for each value
// It returns the subdag that will be executed for each value
// If vertex already exist it will panic
// When a vertex is dag, operations are ommited.
// foreach: Foreach function
// options: Aggregator
func (this *DagFlow) AddForEachBranch(vertex string, foreach sdk.ForEach, options ...BranchOption) (dag *DagFlow) {
	node := this.udag.GetNode(vertex)
	if node != nil {
		panic(fmt.Sprintf("Error at AddForEachBranch for %s, vertex already exists", vertex))
	}
	node = this.udag.AddVertex(vertex, []*sdk.Operation{})

	if foreach == nil {
		panic(fmt.Sprintf("Error at AddForEachBranch for %s, foreach function not specified", vertex))
	}
	node.AddForEach(foreach)

	for _, option := range options {
		o := &BranchOptions{}
		o.reset()
		option(o)
		if o.aggregator != nil {
			node.AddSubAggregator(o.aggregator)
		}
		if o.noforwarder == true {
			node.AddForwarder("dynamic", nil)
		}
	}

	dag = CreateDag()
	err := node.AddForEachDag(dag.udag)
	if err != nil {
		panic(fmt.Sprintf("Error at AddForEachBranch for %s, %v", vertex, err))
	}
	return
}

// AddConditionalBranch composites multiple dags as a subdag which executes for a conditions matched
// and returns the set of dags based on the condition passed
// If vertex already exist it will override the existing definition,
// If not new vertex will be created.
// When a vertex is dag, operations are ommited.
// condition: Condition function
// options: Aggregator
func (this *DagFlow) AddConditionalBranch(vertex string, conditions []string,
	condition sdk.Condition, options ...BranchOption) (conditiondags map[string]*DagFlow) {

	node := this.udag.GetNode(vertex)
	if node != nil {
		panic(fmt.Sprintf("Error at AddForEachBranch for %s, vertex already exists", vertex))
	}
	node = this.udag.AddVertex(vertex, []*sdk.Operation{})
	if condition == nil {
		panic(fmt.Sprintf("Error at AddConditionalBranch for %s, condition function not specified", vertex))
	}
	node.AddCondition(condition)

	for _, option := range options {
		o := &BranchOptions{}
		o.reset()
		option(o)
		if o.aggregator != nil {
			node.AddSubAggregator(o.aggregator)
		}
		if o.noforwarder == true {
			node.AddForwarder("dynamic", nil)
		}
	}
	conditiondags = make(map[string]*DagFlow)
	for _, conditionKey := range conditions {
		dag := CreateDag()
		node.AddConditionalDag(conditionKey, dag.udag)
		conditiondags[conditionKey] = dag
	}
	return
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
func (this *DagFlow) AddEdge(from, to string, opts ...Option) {
	err := this.udag.AddEdge(from, to)
	if err != nil {
		panic(fmt.Sprintf("Error at AddEdge for %s-%s, %v", from, to, err))
	}
	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if o.noforwarder == true {
			fromNode := this.udag.GetNode(from)
			// Add a nil forwarder overriding the default forwarder
			fromNode.AddForwarder(to, nil)
		}

		// in case there is a override
		if o.forwarder != nil {
			fromNode := this.udag.GetNode(from)
			fromNode.AddForwarder(to, o.forwarder)
		}
	}
}
