package main

import (
	"bytes"
	"fmt"
	"github.com/rs/xid"
	"github.com/s8sg/faaschain"
	"github.com/s8sg/faaschain/sdk"
	"handler/function"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	chainName = ""
)

func getGateway() string {
	gateway := os.Getenv("gateway")
	if gateway == "" {
		gateway = "gateway:8080"
	}
	return gateway
}

func getChainName() string {
	return os.Getenv("chain_name")
}

func makeQueryStringFromParam(params map[string][]string) string {
	if params == nil {
		return ""
	}
	result := ""
	for key, array := range params {
		for _, value := range array {
			keyVal := fmt.Sprintf("%s-%s", key, value)
			if result == "" {
				result = "?" + keyVal
			} else {
				result = result + "&" + keyVal
			}
		}
	}
	return result
}

// build upstream request for function
func buildUpstreamRequest(function string, data []byte, params map[string][]string, headers map[string]string) *http.Request {
	url := "http://" + function + ":8080"
	queryString := makeQueryStringFromParam(params)
	if queryString != "" {
		url = url + queryString
	}

	var method string

	if method, ok := headers["method"]; !ok {
		method = os.Getenv("default-method")
		if method == "" {
			method = "POST"
		}
	}

	httpreq, _ := http.NewRequest(method, url, bytes.NewBuffer(data))

	for key, value := range headers {
		httpreq.Header.Set(key, value)
	}

	return httpreq
}

// build upstream request for callback
func buildUpstreamRequestCallback(callbackUrl string, data []byte, params map[string][]string, headers map[string]string) *http.Request {
	url := callbackUrl
	queryString := makeQueryStringFromParam(params)
	if queryString != "" {
		url = url + queryString
	}

	var method string

	if method, ok := headers["method"]; !ok {
		method = os.Getenv("default-method")
		if method == "" {
			method = "POST"
		}
	}

	httpreq, _ := http.NewRequest(method, url, bytes.NewBuffer(data))

	for key, value := range headers {
		httpreq.Header.Set(key, value)
	}

	return httpreq
}

// builds a chain from request
func buildChain(data []byte) (fchain *faaschain.Fchain, requestData []byte) {

	requestId := ""

	// decode the request to find if chain definition exists
	request, err := sdk.DecodeRequest(data)
	if err != nil {
		fchain = faaschain.NewFaaschain("http://"+getGateway(), chainName)
		// Generate request Id
		requestId = xid.New().String()
		log.Printf("[request `%s`] Created", requestId)
		// Set execution Pos
		fchain.GetChain().ExecutionPosition = 0
		requestData = data

		// trace req - mark as start of req
		startReqSpan(requestId)
	} else {
		// Get the request ID
		requestId = request.GetID()
		log.Printf("[Request `%s`] Received", requestId)
		chain, err := request.GetChain()
		if err != nil {
			log.Fatalf("[request `%s`] Failed, error : Failed to parse chain def, %v", requestId, err)
		}
		// Get execution position
		executionPos := chain.ExecutionPosition

		// Override the old chain with new
		fchain = faaschain.NewFaaschain("http://"+getGateway(), chainName)
		// Set execution Pos
		fchain.GetChain().ExecutionPosition = executionPos
		requestData = request.GetData()

		// Continue request span
		continueReqSpan(requestId)
	}
	// set request ID
	fchain.SetId(requestId)

	return
}

// Execute functions for current phase
func execute(fchain *faaschain.Fchain, request []byte) ([]byte, error) {
	var result []byte
	var err error
	var httpreq *http.Request

	chain := fchain.GetChain()

	phase := chain.GetCurrentPhase()

	// trace phase - mark as start of phase
	startPhaseSpan(chain.ExecutionPosition, fchain.GetId())

	// Execute all function
	for _, function := range phase.GetFunctions() {

		switch {
		// If function
		case function.GetName() != "":
			name := function.GetName()
			params := function.GetParams()
			headers := function.GetHeaders()

			// Check if intermidiate data
			if result == nil {
				httpreq = buildUpstreamRequest(name, request, params, headers)
			} else {
				httpreq = buildUpstreamRequest(name, result, params, headers)
			}
			client := &http.Client{}
			resp, err := client.Do(httpreq)
			if err != nil {
				err = fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %v", chain.ExecutionPosition, name, err)
				return nil, err
			}
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return nil, fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %s", chain.ExecutionPosition, name, resp.Status)
			}
			// read Response
			result, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %v", chain.ExecutionPosition, name, err)
			}
		// If callback
		case function.CallbackUrl != "":
			cburl := function.CallbackUrl
			params := function.GetParams()
			headers := function.GetHeaders()

			// Check if intermidiate data
			if result == nil {
				httpreq = buildUpstreamRequestCallback(cburl, request, params, headers)
			} else {
				httpreq = buildUpstreamRequestCallback(cburl, result, params, headers)
			}
			client := &http.Client{}
			resp, err := client.Do(httpreq)
			cbresult, _ := ioutil.ReadAll(resp.Body)
			if err != nil {
				err = fmt.Errorf("Phase(%d), Callback(%s), error: callback failed, %v %s", chain.ExecutionPosition, cburl, err, string(cbresult))
				return nil, err
			}
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				log.Printf("Phase(%d), Callback(%s), error: callback failed, %s, %s", chain.ExecutionPosition, cburl, resp.Status, string(cbresult))
			}
		// If modifier
		default:
			// Check if intermidiate data
			if result == nil {
				result, err = function.Mod(request)
			} else {
				result, err = function.Mod(result)
			}
			if err != nil {
				return nil, fmt.Errorf("Phase(%d), error: Failed at modifier, %v", chain.ExecutionPosition, err)
			}
			if result == nil {
				result = []byte("")
			}
		}
	}

	// trace phase - mark as end of phase
	stopPhaseSpan(chain.ExecutionPosition)

	// Update execution position
	chain.UpdateExecutionPosition()

	return result, nil
}

func forwardAsync(fchain *faaschain.Fchain, result []byte) ([]byte, error) {

	// get chain
	chain := fchain.GetChain()

	// get request id
	id := fchain.GetId()

	// Build current chain definition
	chaindef, _ := chain.Encode()
	// Build request
	uprequest := sdk.BuildRequest(id, string(chaindef), result)
	// Make request data
	data, _ := uprequest.Encode()

	// build url for calling the chain in async
	httpreq, _ := http.NewRequest(http.MethodPost, fchain.GetAsyncUrl(), bytes.NewReader(data))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")

	// extend req span for async call
	extendReqSpan(chain.ExecutionPosition-1, fchain.GetAsyncUrl(), httpreq)

	client := &http.Client{}
	res, resErr := client.Do(httpreq)
	resdata, _ := ioutil.ReadAll(res.Body)
	if resErr != nil {
		return resdata, resErr
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return resdata, fmt.Errorf(res.Status)
	}
	return resdata, nil
}

// Handle request Response
func handleResponse(fchain *faaschain.Fchain, result []byte) ([]byte, error) {

	// get chain
	chain := fchain.GetChain()

	switch {

	// If chain has only one phase or if the chain has completed excution return
	case chain.CountPhases() == 1 || chain.GetCurrentPhase() == nil:
		log.Printf("[request `%s`] Finished successfully ", fchain.GetId())

		return result, nil

	// In default case we forward the partial chain to same chain-function
	default:
		// forward the chain request
		resp, forwardErr := forwardAsync(fchain, result)
		if forwardErr != nil {
			return nil, fmt.Errorf("Phase(%d): error: %v, %s, url %s",
				chain.ExecutionPosition, forwardErr,
				string(resp), fchain.GetAsyncUrl())
		}

		log.Printf("[Request `%s`] Phase %d request submitted",
			fchain.GetId(), chain.ExecutionPosition)
	}

	return []byte(""), nil
}

func handleChain(data []byte) string {

	var resp []byte

	// Get chain name
	chainName = getChainName()
	if chainName == "" {
		fmt.Errorf("Error: chain name must be provided when deployed as function")
		os.Exit(1)
	}

	// initialize traceserve if tracing enabled
	err := initGlobalTracer(chainName)
	if err != nil {
		log.Printf(err.Error())
	}

	// Chain Execution Steps
	{
		// BUILD: build the chain based on execution request
		fchain, data := buildChain(data)

		// DEFINE: Get Chain definition from user implemented Define()
		err := function.Define(fchain)
		if err != nil {
			log.Fatalf("[request `%s`] Failed, %v", fchain.GetId(), err)
		}

		// EXECUTE: execute the chain based on current phase
		result, err := execute(fchain, data)
		if err != nil {
			log.Fatalf("[request `%s`] Failed, %v", fchain.GetId(), err)
		}

		// HANDLE: Handle the execution state of last phase
		resp, err = handleResponse(fchain, result)
		if err != nil {
			log.Fatalf("[request `%s`] Failed, %v", fchain.GetId(), err)
		}
	}

	// stop req span if request has finished
	stopReqSpan()

	// flash any pending trace item if tracing enabled
	flushTracer()

	return string(resp)
}
