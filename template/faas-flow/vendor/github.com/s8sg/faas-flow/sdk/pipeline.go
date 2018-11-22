package sdk

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	TYPE_CHAIN = "chain"
	TYPE_DAG   = "dag"
)

// PipelineErrorHandler the error handler OnFailure() registration on pipeline
type PipelineErrorHandler func(error) ([]byte, error)

// PipelineHandler definition for the Finally() registration on pipeline
type PipelineHandler func(string)

type Pipeline struct {
	PipelineType string `json:"type"` // Pipeline type denotes a dag or a chain

	Dag                  *Dag   `json:"-"`            // Dag that will be executed
	DagExecutionPosition string `json:"dag-position"` // Denotes the node that is executing now

	FailureHandler PipelineErrorHandler `json:"-"`
	Finally        PipelineHandler      `json:"-"`
}

func CreatePipeline() *Pipeline {
	pipeline := &Pipeline{}
	pipeline.PipelineType = TYPE_CHAIN
	pipeline.Dag = NewDag()
	pipeline.DagExecutionPosition = ""
	return pipeline
}

// Node
func (pipeline *Pipeline) CountNodes() int {
	return len(pipeline.Dag.nodes)
}

func (pipeline *Pipeline) GetNextNodes() []*Node {
	return pipeline.GetCurrentNode().Children()
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

func (pipeline *Pipeline) IsInitialNode() bool {
	if pipeline.GetCurrentNode().indegree == 0 {
		return true
	}
	return false
}

func (pipeline *Pipeline) GetInitialNodeId() string {
	node := pipeline.Dag.GetInitialNode()
	if node != nil {
		return node.Id
	}
	return "0"
}

func (pipeline *Pipeline) GetCurrentNode() *Node {
	return pipeline.Dag.GetNode(pipeline.DagExecutionPosition)
}

func (pipeline *Pipeline) UpdateDagExecutionPosition(vertex string) {
	pipeline.DagExecutionPosition = vertex
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
	pipeline.DagExecutionPosition = temp.DagExecutionPosition
	pipeline.PipelineType = temp.PipelineType
}

// MakeDotGraph create a dot graph of the pipeline
func (pipeline *Pipeline) MakeDotGraph() string {
	var sb strings.Builder
	sb.WriteString("digraph depgraph {\n\trankdir=LR;\n")
	for _, node := range pipeline.Dag.nodes {
		if len(node.children) == 0 {
			sb.WriteString(fmt.Sprintf("\t\"%s\";\n", node.Id))
			continue
		}
		modifierType := ""
		for _, operation := range node.Operations() {
			switch {
			case operation.Function != "":
				modifierType += " func:" + operation.Function
			case operation.CallbackUrl != "":
				modifierType += " callback:" + operation.CallbackUrl
			default:
				modifierType += " modifier"
			}
		}

		for _, child := range node.children {
			sb.WriteString(fmt.Sprintf(`%s -> %s [label="%v"]`, node.Id, child.Id, modifierType))
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}
