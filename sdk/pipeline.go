package sdk

import (
	"encoding/json"
	"fmt"
)

const (
	TYPE_CHAIN = "chain"
	TYPE_DAG   = "dag"
)

const (
	DEPTH_INCREMENT = 1
	DEPTH_DECREMENT = -1
	DEPTH_SAME      = 0
)

// PipelineErrorHandler the error handler OnFailure() registration on pipeline
type PipelineErrorHandler func(error) ([]byte, error)

// PipelineHandler definition for the Finally() registration on pipeline
type PipelineHandler func(string)

type Pipeline struct {
	PipelineType string `json:"type"` // Pipeline type denotes a dag or a chain

	Dag *Dag `json:"-"` // Dag that will be executed

	ExecutionPosition map[string]string `json:"pipeline-execution-position"` // Denotes the node that is executing now
	ExecutionDepth    int               `json:"pipeline-execution-depth"`    // Denotes the depth of subgraph its executing

	DynamicDependencyCount map[string]int `json:"pipeline-dynamic-dependency-count"` // Denotes the no of dependency for a node

	FailureHandler PipelineErrorHandler `json:"-"`
	Finally        PipelineHandler      `json:"-"`
}

// CreatePipeline creates a faasflow pipeline
func CreatePipeline(name string) *Pipeline {
	pipeline := &Pipeline{}
	pipeline.PipelineType = TYPE_CHAIN
	pipeline.Dag = NewDag()
	// For default dag set the Id to flow-name
	pipeline.Dag.Id = name
	pipeline.ExecutionPosition = make(map[string]string, 0)
	pipeline.ExecutionDepth = 0
	return pipeline
}

// CountNodes counts the no of node added in the Pipeline Dag.
// It doesn't count subdags node
func (pipeline *Pipeline) CountNodes() int {
	return len(pipeline.Dag.nodes)
}

// GetAllNodesId returns a recursive list of all nodes that belongs to the pipeline
func (pipeline *Pipeline) GetAllNodesId() []string {
	nodes := pipeline.Dag.GetNodes()
	return nodes
}

// GetInitialNodeId Get the very first node of the pipeline
func (pipeline *Pipeline) GetInitialNodeId() string {
	node := pipeline.Dag.GetInitialNode()
	if node != nil {
		return node.Id
	}
	return "0"
}

// GetCurrentNodeDag returns the current node and current dag based on execution position
func (pipeline *Pipeline) GetCurrentNodeDag() (*Node, *Dag) {
	index := 0
	dag := pipeline.Dag
	indexStr := ""
	for index < pipeline.ExecutionDepth {
		indexStr = fmt.Sprintf("%d", index)
		node := dag.GetNode(pipeline.ExecutionPosition[indexStr])
		dag = node.subDag
		index++
	}
	indexStr = fmt.Sprintf("%d", index)
	node := dag.GetNode(pipeline.ExecutionPosition[indexStr])
	return node, dag
}

// UpdatePipelineExecutionPosition updates pipeline execution position
// specifyed depthAdjustment and vertex denotes how the ExecutionPosition must be altered
func (pipeline *Pipeline) UpdatePipelineExecutionPosition(depthAdjustment int, vertex string) {
	pipeline.ExecutionDepth = pipeline.ExecutionDepth + depthAdjustment
	depthStr := fmt.Sprintf("%d", pipeline.ExecutionDepth)
	pipeline.ExecutionPosition[depthStr] = vertex
}

// SetDag overrides the default dag
func (pipeline *Pipeline) SetDag(dag *Dag) {
	pipeline.Dag = dag
	pipeline.PipelineType = TYPE_DAG
}

// decodePipeline decodes a json marshaled pipeline
func decodePipeline(data []byte) (*Pipeline, error) {
	pipeline := &Pipeline{}
	err := json.Unmarshal(data, pipeline)
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}

// GetState get a state of a pipeline by encoding in JSON
func (pipeline *Pipeline) GetState() string {
	encode, _ := json.Marshal(pipeline)
	return string(encode)
}

// ApplyState apply a state to a pipeline by from encoded JSON pipeline
func (pipeline *Pipeline) ApplyState(state string) {
	temp, _ := decodePipeline([]byte(state))
	pipeline.ExecutionDepth = temp.ExecutionDepth
	pipeline.ExecutionPosition = temp.ExecutionPosition
	pipeline.PipelineType = temp.PipelineType
}
