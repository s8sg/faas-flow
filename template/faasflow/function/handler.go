package function

import (
	"fmt"
	"github.com/s8sg/faasflow"
)

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
	flow.Modify(func(data []byte) ([]byte, error) {
		return []byte(fmt.Sprintf("you said \"%s\"", string(data))), nil
	})
	return
}
