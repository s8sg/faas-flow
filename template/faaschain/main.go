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
		chainname := getChainName()
		if chainname == "" {
			log.Printf("Chain name must be provided when chain deployed as function")
		}

		fchain = faaschain.NewFaaschain("http://"+getGateway(), chainname)
		// Generate request Id
		requestId = xid.New().String()
		log.Printf("[request `%s`] Created", requestId)
		// Set execution Pos
		fchain.GetChain().ExecutionPosition = 0
		requestData = data
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

		chainname := getChainName()
		if chainname == "" {
			log.Printf("Chain name must be provided when chain deployed as function")
		}

		// Override the old chain with new
		fchain = faaschain.NewFaaschain("http://"+getGateway(), chainname)
		// Set execution Pos
		fchain.GetChain().ExecutionPosition = executionPos
		requestData = request.GetData()
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
				err = fmt.Errorf("Phase(%d), Function(%s), error : Failed to execute function %v", chain.ExecutionPosition, name, err)
				return nil, err
			}
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return nil, fmt.Errorf("Phase(%d), Function(%s), error : Failed to execute function %s", chain.ExecutionPosition, name, resp.Status)
			}
			// read Response
			result, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Phase(%d), Function(%s), error : Failed to read result %v", chain.ExecutionPosition, name, err)
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
				err = fmt.Errorf("Phase(%d), Callback(%s), error : Failed to execute callback %v, %s", chain.ExecutionPosition, cburl, err, string(cbresult))
				return nil, err
			}
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				log.Printf("Phase(%d), Callback(%s), error : Failed to execute callback %s, %s", chain.ExecutionPosition, cburl, resp.Status, string(cbresult))
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
				return nil, fmt.Errorf("Phase(%d), error : Failed to execute modifier %v", chain.ExecutionPosition, err)
			}
			if result == nil {
				result = []byte("")
			}
		}
	}

	// Update execution position
	chain.UpdateExecutionPosition()

	return result, nil
}

// Handle request Response
func handleResponse(fchain *faaschain.Fchain, result []byte) ([]byte, error) {

	// get request Id
	id := fchain.GetId()

	// get chain
	chain := fchain.GetChain()

	// for a single phase just return the result to the caller
	// Single phase means no async call has been made by the chain
	// In that case if chain is executed in async mode the x-callback-url
	// will be handled by the caller
	if chain.CountPhases() == 1 {
		return result, nil
	}

	// If chain has already finished execution
	// Then return the result which will be handled by the "X-Callback-Url" If provided
	// TODO: Allow Callback Call in Chain definition
	if chain.GetCurrentPhase() == nil {
		return result, nil
	}

	// If the chain has phases left to handle
	// Create new request with chain info and
	// Build current chain definition
	chaindef, _ := chain.Encode()
	// Build request
	uprequest := sdk.BuildRequest(id, string(chaindef), result)
	// Make request data
	data, _ := uprequest.Encode()

	// build url for async function
	httpreq, _ := http.NewRequest(http.MethodPost, fchain.GetAsyncUrl(), bytes.NewReader(data))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")

	// If the chain has only one more phase to execute and callback URL was provided
	// Execute the last phase with the callback URL so that the returned result gets
	// handled via openfaas callback
	/*
		if chain.IsLastPhase() {
			if chain.CallbackUrl != "" {
				httpreq.Header.Add("X-Callback-Url", chain.CallbackUrl)
			}
		}*/

	client := &http.Client{}
	res, resErr := client.Do(httpreq)
	resdata, _ := ioutil.ReadAll(res.Body)
	if resErr != nil {
		return nil, fmt.Errorf("Phase(%d): error %v, %s, url %s", chain.ExecutionPosition, resErr, string(resdata), fchain.GetAsyncUrl())
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("Phase(%d): error %s, %s, url %s", chain.ExecutionPosition, res.Status, string(resdata), fchain.GetAsyncUrl())
	}

	log.Printf("Execution request for phase %d has been successfully submitted", chain.ExecutionPosition)

	return []byte(""), nil
}

func handle(data []byte) string {

	// BUILD: build the chain based on execution request
	log.Printf("building chain request")
	fchain, data := buildChain(data)

	// DEFINE: Get Chain definition from user implemented Define()
	log.Printf("generating function defintion")
	err := function.Define(fchain)
	if err != nil {
		log.Fatalf("[request `%s`] Failed, %v", fchain.GetId(), err)
	}

	// Execute: execute the chain based on current phase
	log.Printf("executing phase %d", fchain.GetChain().ExecutionPosition)
	result, err := execute(fchain, data)
	if err != nil {
		log.Fatalf("[request `%s`] Failed, %v", fchain.GetId(), err)
	}

	// Handle: handle a chain response
	log.Printf("handling response for %d", fchain.GetChain().ExecutionPosition)
	// Handle a response of FaaSChain Function
	resp, err := handleResponse(fchain, result)
	if err != nil {
		log.Fatalf("[request `%s`] Failed, %v", fchain.GetId(), err)
	}

	return string(resp)
}

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read standard input: %s", err.Error())
	}

	fmt.Println(handle(input))
}
