package function

import (
	faasflow "github.com/s8sg/faas-flow"
)

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
	flow.Apply("func1", faasflow.Sync).Apply("func2", faasflow.Sync)
	return
}
