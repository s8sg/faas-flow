package handler

import (
	"errors"
	"fmt"
	"handler/runtime/controller/util"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/faasflow/sdk/executor"
)

func PartialExecuteFlowHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Requested %s %s\n", req.Method, req.URL)
	log.Printf("Executing flow %s\n", id)

	var stateOption executor.ExecutionStateOption

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	callbackURL := req.Header.Get(util.CallbackUrlHeader)

	if id == "" {
		return nil, errors.New("request ID not set in partial request")
	}
	partialState, err := executor.DecodePartialReq(body)
	if err != nil {
		return nil, errors.New("failed to decode partial state")
	}
	stateOption = executor.PartialRequest(partialState)

	// Create a flow executor with provided executor
	flowExecutor := executor.CreateFlowExecutor(ex, nil)
	resp, err := flowExecutor.Execute(stateOption)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request. %s", err.Error())
	}
	headers := w.Header()
	headers[util.RequestIdHeader] = []string{flowExecutor.GetReqId()}
	headers[util.CallbackUrlHeader] = []string{callbackURL}

	return resp, nil
}
