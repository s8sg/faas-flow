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

	Phases            []*Phase `json:"-"`        // Phases that will be executed in async
	ExecutionPosition int      `json:"position"` // Position of Executor

	Dag                  *Dag   `json:"-"`            // Dag that will be executed
	DagExecutionPosition string `json:"dag-position"` // Position of Executor in Dag

	FailureHandler PipelineErrorHandler `json:"-"`
	Finally        PipelineHandler      `json:"-"`
}

func CreatePipeline() *Pipeline {
	pipeline := &Pipeline{}
	pipeline.PipelineType = TYPE_CHAIN
	pipeline.Phases = make([]*Phase, 0)
	pipeline.ExecutionPosition = 0
	pipeline.Dag = nil
	pipeline.DagExecutionPosition = ""
	return pipeline
}

// Phase
func (pipeline *Pipeline) CountPhases() int {
	return len(pipeline.Phases)
}

func (pipeline *Pipeline) AddPhase(phase *Phase) {
	pipeline.Phases = append(pipeline.Phases, phase)
}

func (pipeline *Pipeline) GetCurrentPhase() *Phase {
	if pipeline.ExecutionPosition < len(pipeline.Phases) {
		return pipeline.Phases[pipeline.ExecutionPosition]
	} else {
		return nil
	}
}

func (pipeline *Pipeline) IsLastPhase() bool {
	return ((len(pipeline.Phases) - 1) == pipeline.ExecutionPosition)
}

func (pipeline *Pipeline) GetLastPhase() *Phase {
	phaseCount := len(pipeline.Phases)
	return pipeline.Phases[phaseCount-1]
}

func (pipeline *Pipeline) UpdateExecutionPosition() {
	pipeline.ExecutionPosition = pipeline.ExecutionPosition + 1
}

// Node
func (pipeline *Pipeline) CountNodes() int {
	return len(pipeline.Dag.nodes)
}

func (pipeline *Pipeline) GetNextNodes() []*Node {
	return pipeline.GetCurrentNode().Children()
}

func (pipeline *Pipeline) IsInitialNode() bool {
	if pipeline.GetCurrentNode().indegree == 0 {
		return true
	}
	return false
}

func (pipeline *Pipeline) GetCurrentNode() *Node {
	return pipeline.Dag.Node(pipeline.DagExecutionPosition)
}

func (pipeline *Pipeline) UpdateDagExecutionPosition(vertex string) {
	pipeline.DagExecutionPosition = vertex
}

func (pipeline *Pipeline) SetDag(dag *Dag) {
	pipeline.Dag = dag
	pipeline.PipelineType = TYPE_DAG
}

// Common
func (pipeline *Pipeline) Encode() ([]byte, error) {
	return json.Marshal(pipeline)
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
	temp := *pipeline
	temp.Phases = nil
	encode, _ := json.Marshal(&temp)
	return string(encode)
}

func (pipeline *Pipeline) ApplyState(state string) {
	var temp Pipeline
	json.Unmarshal([]byte(state), &temp)
	pipeline.ExecutionPosition = temp.ExecutionPosition
	pipeline.DagExecutionPosition = temp.DagExecutionPosition
	pipeline.PipelineType = temp.PipelineType
}

// MakeDotGraph create a dot graph of the pipeline
func (pipeline *Pipeline) MakeDotGraph() string {
	switch pipeline.PipelineType {
	case TYPE_DAG:
		return pipeline.Dag.makeDagDotGraph()
	case TYPE_CHAIN:
		return pipeline.makeChainDotGraoh()
	}
	return ""
}

func (this *Pipeline) makeChainDotGraoh() string {
	var sb strings.Builder
	sb.WriteString("digraph depgraph {\n\trankdir=LR;\n")
	for index, phase := range this.Phases {
		if index == len(this.Phases)-1 {
			sb.WriteString(fmt.Sprintf("\t\"phase-%d\";\n", index+1))
			continue
		}
		modifierTypes := ""
		for _, function := range phase.Functions {
			switch {
			case function.GetName() != "":
				modifierTypes += " func:" + function.GetName()
			case function.CallbackUrl != "":
				modifierTypes += " callback:" + function.CallbackUrl
			default:
				modifierTypes += " modifier"
			}
		}
		sb.WriteString(fmt.Sprintf(`phase-%d -> phase-%d [label="%s"]`, index+1, index+2, modifierTypes))
	}
	sb.WriteString("}\n")
	return sb.String()
}

func (this *Dag) makeDagDotGraph() string {
	var sb strings.Builder
	sb.WriteString("digraph depgraph {\n\trankdir=LR;\n")
	for _, node := range this.nodes {
		if len(node.children) == 0 {
			sb.WriteString(fmt.Sprintf("\t\"%s\";\n", node.Id))
			continue
		}
		modifierType := ""
		switch {
		case node.val.GetName() != "":
			modifierType += " func:" + node.val.GetName()
		case node.val.CallbackUrl != "":
			modifierType += " callback:" + node.val.CallbackUrl
		default:
			modifierType += " modifier"
		}

		for _, child := range node.children {
			sb.WriteString(fmt.Sprintf(`%s -> %s [label="%v"]`, node.Id, child.Id, modifierType))
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}
