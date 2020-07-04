package controller

import (
	"handler/eventhandler"
	"handler/openfaas"
	"net/http"

	"github.com/faasflow/sdk/executor"
)

func newRequestExecutor(request *http.Request) (executor.Executor, error) {
	eventHandler := &eventhandler.FaasEventHandler{}
	ex := &openfaas.OpenFaasExecutor{StateStore: stateStore, DataStore: dataStore, EventHandler: eventHandler}

	err := ex.Init(request)
	if err != nil {
		return nil, err
	}

	return ex, nil
}
