package function

import (
	"fmt"
	faasflow "github.com/s8sg/faas-flow"
)

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
	flow.SyncNode().Modify(func(data []byte) ([]byte, error) {
		return []byte(fmt.Sprintf("you said \"%s\"", string(data))), nil
	})
	return
}

// DefineStateStore provides the override of the default StateStore
func DefineStateStore() (faasflow.StateStore, error) {
	return nil, nil
}

// ProvideDataStore provides the override of the default DataStore
func DefineDataStore() (faasflow.DataStore, error) {
	return nil, nil
}
