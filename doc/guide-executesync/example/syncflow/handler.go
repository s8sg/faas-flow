package function

import (
	"github.com/s8sg/faasflow"
)

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
	flow.Apply("func1", faasflow.Sync).Apply("func2", faasflow.Sync)
	return
}
