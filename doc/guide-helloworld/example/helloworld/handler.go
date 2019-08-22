package function

import (
	"fmt"

	faasflow "github.com/s8sg/faas-flow"
)

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
	flow.Modify(func(data []byte) ([]byte, error) {
		return []byte(fmt.Sprintf("Hello World! You said %s!", string(data))), nil
	})
	return
}
