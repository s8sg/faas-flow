package handler

import (
	"fmt"
	"handler/runtime/controller/util"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/faasflow/sdk/executor"
)

func ExecuteFlowHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Requested %s %s\n", req.Method, req.URL)
	log.Printf("Executing flow %s\n", id)

	var stateOption executor.ExecutionStateOption

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	callbackURL := req.Header.Get(util.CallbackUrlHeader)
	rawRequest := &executor.RawRequest{}
	rawRequest.Data = body
	rawRequest.Query = req.URL.RawQuery
	rawRequest.AuthSignature = req.Header.Get(util.AuthSignatureHeader)
	if id != "" {
		rawRequest.RequestId = id
	}
	stateOption = executor.NewRequest(rawRequest)

	// Create a flow controller, OpenFaaSExecutor implements controller
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
