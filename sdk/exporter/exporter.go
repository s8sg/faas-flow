package exporter

import (
	"fmt"
	"github.com/s8sg/faas-flow/sdk"
)

// Exporter
type Exporter interface {
	// GetFlowName get nbame of the flow
	GetFlowName() string
	// GetFlowDefinition get definition of the faas-flow
	GetFlowDefinition(*sdk.Pipeline, *sdk.Context) error
}

// FlowExporter faasflow exporter
type FlowExporter struct {
	flow     *sdk.Pipeline
	flowName string
	exporter Exporter // exporter
}

// createContext create a context from request handler
func (fexp *FlowExporter) createContext() *sdk.Context {
	context := sdk.CreateContext("export", "",
		fexp.flowName, nil)

	return context
}

// Export retrieve faasflow definition
func (fexp *FlowExporter) Export() ([]byte, error) {

	// Init flow
	fexp.flow = sdk.CreatePipeline()
	fexp.flowName = fexp.exporter.GetFlowName()

	context := fexp.createContext()

	// Get definition: Get Pipeline definition from user implemented Define()
	err := fexp.exporter.GetFlowDefinition(fexp.flow, context)
	if err != nil {
		return nil, fmt.Errorf("Failed to define flow, %v", err)
	}

	definition := sdk.GetPipelineDefinition(fexp.flow)

	return []byte(definition), nil
}

// CreateFlowExporter initiate a FlowExporter with a provided Executor
func CreateFlowExporter(exporter Exporter) (fexp *FlowExporter) {
	fexp = &FlowExporter{}
	fexp.exporter = exporter

	return fexp
}
