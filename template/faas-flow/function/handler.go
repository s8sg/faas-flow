package function

import (
	"fmt"
	faasflow "github.com/s8sg/faas-flow"
	sdk "github.com/s8sg/faas-flow/sdk"
)

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *sdk.Context) (err error) {
	flow.SyncNode().Modify(func(data []byte) ([]byte, error) {
		return []byte(fmt.Sprintf("you said \"%s\"", string(data))), nil
	})
	return
}

// DefineStateStore provides the override of the default StateStore
func DefineStateStore() (sdk.StateStore, error) {
	return nil, nil
}

// ProvideDataStore provides the override of the default DataStore
func DefineDataStore() (sdk.DataStore, error) {
	return nil, nil
}
