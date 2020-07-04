package controller

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/faasflow/sdk/executor"
)

const (
	CallbackUrlHeader   = "X-Faas-Flow-Callback-Url"
	RequestIdHeader     = "X-Faas-Flow-Reqid"
	AuthSignatureHeader = "X-Hub-Signature"
)

func executeFlowHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Requested %s %s\n", req.Method, req.URL)
	log.Printf("Executing flow %s\n", id)

	var stateOption executor.ExecutionStateOption

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	state := req.Header.Get(RequestIdHeader)
	callbackURL := req.Header.Get(CallbackUrlHeader)
	if state == "" {
		rawRequest := &executor.RawRequest{}
		rawRequest.Data = body
		rawRequest.Query = req.URL.RawQuery
		rawRequest.AuthSignature = req.Header.Get(AuthSignatureHeader)
		if id != "" {
			rawRequest.RequestId = id
		}
		stateOption = executor.NewRequest(rawRequest)

	} else {
		if id == "" {
			return nil, errors.New("request ID not set in partial request")
		}
		partialState, err := executor.DecodePartialReq(body)
		if err != nil {
			return nil, errors.New("failed to decode partial state")
		}
		stateOption = executor.PartialRequest(partialState)
	}

	// Create a flow controller, OpenFaaSExecutor implements controller
	flowExecutor := executor.CreateFlowExecutor(ex, nil)
	resp, err := flowExecutor.Execute(stateOption)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request. %s", err.Error())
	}
	headers := w.Header()
	headers[RequestIdHeader] = []string{flowExecutor.GetReqId()}
	headers[CallbackUrlHeader] = []string{callbackURL}

	return resp, nil
}
