package executor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/s8sg/faas-flow/sdk/executor"
)

func executeFlowHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Requested %s %s\n", req.Method, req.URL)
	log.Printf("Executing flow %s\n", id)

	var stateOption executor.ExecutionStateOption

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	// TODO: fix this
	openFaasEx := ex.(*openFaasExecutor)

	state := req.Header.Get("X-Faas-Flow-State")
	callbackURL := req.Header.Get("X-Faas-Flow-Callback-Url")
	openFaasEx.callbackURL = callbackURL
	if state == "" {
		rawRequest := &executor.RawRequest{}
		rawRequest.Data = body
		rawRequest.Query = req.URL.RawQuery
		rawRequest.AuthSignature = req.Header.Get("X-Hub-Signature")
		// Check if any request Id is passed
		if id != "" {
			rawRequest.RequestId = id
		}
		stateOption = executor.NewRequest(rawRequest)

	} else {
		if id == "" {
			return nil, errors.New("request ID not set in partial request")
		}

		openFaasEx.openFaasEventHandler.header = req.Header
		partialState, err := executor.DecodePartialReq(body)
		if err != nil {
			return nil, errors.New("failed to decode partial state")
		}
		stateOption = executor.PartialRequest(partialState)
	}

	// Create a flow executor, OpenFaaSExecutor implements executor
	flowExecutor := executor.CreateFlowExecutor(ex, nil)
	resp, err := flowExecutor.Execute(stateOption)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request. %s", err.Error())
	}
	headers := w.Header()
	headers["X-Faas-Flow-Reqid"] = []string{flowExecutor.GetReqId()}
	headers["X-Faas-Flow-Callback-Url"] = []string{callbackURL}

	return resp, nil
}
