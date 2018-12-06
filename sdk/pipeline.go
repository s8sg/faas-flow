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

	FailureHandler PipelineErrorHandler `json:"-"`
	Finally        PipelineHandler      `json:"-"`
}

func CreatePipeline(name string) *Pipeline {
	pipeline := &Pipeline{}
	pipeline.PipelineType = TYPE_CHAIN
	pipeline.Dag = NewDag()
	pipeline.Dag.Id = name
	pipeline.ExecutionPosition = make(map[string]string, 0)
	pipeline.ExecutionDepth = 0
	pipeline.ExecutionPosition["0"] = name
	return pipeline
}

// Node
func (pipeline *Pipeline) CountNodes() int {
	return len(pipeline.Dag.nodes)
}

func (pipeline *Pipeline) GetAllNodesId() []string {
	nodes := make([]string, len(pipeline.Dag.nodes))
	i := 0
	for id, _ := range pipeline.Dag.nodes {
		nodes[i] = id
		i++
	}
	return nodes
}

func (pipeline *Pipeline) GetInitialNodeId() string {
	node := pipeline.Dag.GetInitialNode()
	if node != nil {
		return node.Id
	}
	return "0"
}

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

func DecodePipeline(data []byte) (*Pipeline, error) {
	pipeline := &Pipeline{}
	err := json.Unmarshal(data, pipeline)
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}

func (pipeline *Pipeline) GetState() string {
	encode, _ := json.Marshal(pipeline)
	return string(encode)
}

func (pipeline *Pipeline) ApplyState(state string) {
	var temp Pipeline
	json.Unmarshal([]byte(state), &temp)
	pipeline.ExecutionDepth = temp.ExecutionDepth
	pipeline.ExecutionPosition = temp.ExecutionPosition
	pipeline.PipelineType = temp.PipelineType
}
