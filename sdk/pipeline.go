package sdk

import (
	"encoding/json"
)

// PipelineErrorHandler the error handler OnFailure() registration on pipeline
type PipelineErrorHandler func(error) ([]byte, error)

// PipelineHandler definition for the Finally() registration on pipeline
type PipelineHandler func(string)

type Pipeline struct {
	Phases            []*Phase `json:"-"`        // Phases that will be executed in async
	ExecutionPosition int      `json:"position"` // Position of Executor

	Dag                  *Dag   `json:"-"`            // Dag that will be executed
	DagExecutionPosition string `json:"dag-position"` // Position of Executor in Dag

	FailureHandler PipelineErrorHandler `json:"-"`
	Finally        PipelineHandler      `json:"-"`
}

func CreatePipeline() *Pipeline {
	pipeline := &Pipeline{}
	pipeline.Phases = make([]*Phase, 0)
	pipeline.ExecutionPosition = 0
	pipeline.Dag = nil
	pipeline.DagExecutionPosition = ""
	return pipeline
}

func DecodePipeline(data []byte) (*Pipeline, error) {
	pipeline := &Pipeline{}
	err := json.Unmarshal(data, pipeline)
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}

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

func (pipeline *Pipeline) UpdateDagExecutionPosition(vertex string) {
	pipeline.DagExecutionPosition = vertex
}

func (pipeline *Pipeline) Encode() ([]byte, error) {
	return json.Marshal(pipeline)
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
}
