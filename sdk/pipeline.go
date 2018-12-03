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

	sb.WriteString("digraph depgraph {")
	sb.WriteString("\n\trankdir=TB;")
	sb.WriteString("\n\tsplines=curved;")
	sb.WriteString("\n\tfontname=\"Courier New\";")
	sb.WriteString("\n\tfontcolor=grey;")

	sb.WriteString("\n\tnode [style=filled fontname=\"Courier\" fontcolor=black]\n")

	// generate nodes
	for _, node := range pipeline.Dag.nodes {

		sb.WriteString(fmt.Sprintf("\n\tsubgraph cluster_%d {", node.index))
		sb.WriteString(fmt.Sprintf("\n\t\tlabel=\"%d-%s\";", node.index, node.Id))
		sb.WriteString("\n\t\tcolor=lightgrey;")
		sb.WriteString("\n\t\tstyle=rounded;\n")

		previousOperation := ""
		for opsindex, operation := range node.Operations() {
			operationStr := ""
			switch {
			case operation.Function != "":
				operationStr = "func-" + operation.Function
			case operation.CallbackUrl != "":
				operationStr = "callback-" +
					operation.CallbackUrl[len(operation.CallbackUrl)-4:]
			default:
				operationStr = "modifier"
			}
			operationKey := fmt.Sprintf("%d.%d-%s", node.index, opsindex+1, operationStr)

			switch {
			case len(node.children) == 0 &&
				opsindex == len(node.Operations())-1:
				sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" [color=pink];",
					operationKey))
			case node.indegree == 0 && opsindex == 0:
				sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" [color=lightblue];",
					operationKey))
			default:
				sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" [color=lightgrey];",
					operationKey))
			}

			if previousOperation != "" {
				sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" -> \"%s\" [label=\"1:1\" color=grey];",
					previousOperation, operationKey))
			}
			previousOperation = operationKey
		}

		sb.WriteString("\n\t}")

		// TODO: Later change to check if 1:N
		relation := "1:1"
		for _, child := range node.children {
			operationStr := ""
			operation := child.Operations()[0]
			switch {
			case operation.Function != "":
				operationStr = "func-" + operation.Function
			case operation.CallbackUrl != "":
				operationStr = "callback-" +
					operation.CallbackUrl[len(operation.CallbackUrl)-4:]
			default:
				operationStr = "modifier"
			}

			childOperationKey := fmt.Sprintf("%d.1-%s",
				child.index, operationStr)

			sb.WriteString(fmt.Sprintf("\n\t\"%s\" -> \"%s\" [label=\"%s\" color=grey];", previousOperation, childOperationKey, relation))
		}

		sb.WriteString("\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}
