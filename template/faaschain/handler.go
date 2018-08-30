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
	"net/url"
	"os"
	"path"
)

var (
	// chainName the name of the chain
	chainName = ""
)

type chainHandler struct {
	fchain   *faaschain.Fchain         // the faaschain
	id       string                    // the request id
	query    string                    // the query string by user
	asyncUrl string                    // the async URL of the chain
	chainDef []byte                    // the chain definition
	finished bool                      // denots the chain has finished execution
	stateM   *requestEmbedStateManager // the deafult statemanager
}

// buildURL builds openfaas function execution url for the chain
func buildURL(gateway, rpath, chain string) string {
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, rpath+"/"+chain)
	return u.String()
}

// newFaaschain creates a new faaschain object
func newChainHandler(gateway string, chain string, id string,
	query string, stateM *requestEmbedStateManager) *chainHandler {

	chandler := &chainHandler{}

	fchain := faaschain.NewFaaschain()

	chandler.fchain = fchain

	chandler.asyncUrl = buildURL(gateway, "async-function", chain)

	chandler.id = id
	chandler.query = query
	chandler.stateM = stateM

	return chandler
}

// getGateway return the gateway address from env
func getGateway() string {
	gateway := os.Getenv("gateway")
	if gateway == "" {
		gateway = "gateway:8080"
	}
	return gateway
}

// getChainName returns the chain name from env
func getChainName() string {
	return os.Getenv("chain_name")
}

// getChain returns the underline fchain.chain object
func (chandler *chainHandler) getChain() *sdk.Chain {
	return chandler.fchain.GetChain()
}

// makeQueryStringFromParam create query string from provided query
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

// buildUpstreamRequest build upstream request for function
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

// buildUpstreamRequestCallback build upstream request for callback
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

// createContext create a context from request handler
func createContext(chandler *chainHandler) *faaschain.Context {
	context := faaschain.CreateContext(chandler.id,
		chandler.getChain().ExecutionPosition+1)
	context.SetStateManager(chandler.stateM)
	context.Query, _ = url.ParseQuery(chandler.query)
	return context
}

// buildChain builds a chain and context from raw request or partial completed request
func buildChain(data []byte) (chandler *chainHandler, requestData []byte) {

	requestId := ""

	// decode the request to find if chain definition exists
	request, err := decodeRequest(data)
	switch {

	case err != nil:
		// Generate request Id
		requestId = xid.New().String()
		log.Printf("[request `%s`] Created", requestId)

		// create chain properties
		query := os.Getenv("Http_Query")
		stateM := createStateManager()

		// Create chandler
		chandler = newChainHandler("http://"+getGateway(), chainName,
			requestId, query, stateM)
		chandler.getChain().ExecutionPosition = 0

		// set request data
		requestData = data

		// trace req - mark as start of req
		startReqSpan(requestId)
	default:
		// Get the request ID
		requestId = request.getID()
		log.Printf("[Request `%s`] Received", requestId)

		// get chain properties
		executionState := request.getExecutionState()
		query := request.getQuery()
		stateM := retriveStateManager(request.getContextState())

		// Create chain and apply execution state
		chandler = newChainHandler("http://"+getGateway(), chainName,
			requestId, query, stateM)
		chandler.getChain().ApplyState(executionState)

		// set request data
		requestData = request.getData()

		// Continue request span
		continueReqSpan(requestId)
	}

	return
}

// executeFunction executes a function call
func executeFunction(chain *sdk.Chain, function *sdk.Function, data []byte) ([]byte, error) {
	var err error

	name := function.GetName()
	params := function.GetParams()
	headers := function.GetHeaders()

	httpreq := buildUpstreamRequest(name, data, params, headers)

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		err = fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %v",
			chain.ExecutionPosition, name, err)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err = fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %s",
			chain.ExecutionPosition, name, resp.Status)
		return nil, err
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %v",
			chain.ExecutionPosition, name, err)
		return nil, err
	}
	return result, nil
}

// executeCallback executes a callback
func executeCallback(chain *sdk.Chain, function *sdk.Function, data []byte) error {
	var err error

	cburl := function.CallbackUrl
	params := function.GetParams()
	headers := function.GetHeaders()

	httpreq := buildUpstreamRequestCallback(cburl, data, params, headers)

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	cbresult, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("Phase(%d), Callback(%s), error: callback failed, %v %s",
			chain.ExecutionPosition, cburl, err, string(cbresult))
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err = fmt.Errorf("Phase(%d), Callback(%s), error: callback failed, %s, %s",
			chain.ExecutionPosition, cburl, resp.Status, string(cbresult))
		return err
	}
	return nil

}

// execute functions for current phase
func execute(chandler *chainHandler, request []byte) ([]byte, error) {
	var result []byte
	var err error

	chain := chandler.getChain()

	phase := chain.GetCurrentPhase()

	log.Printf("[request `%s`] Executing phase %d", chandler.id, chain.ExecutionPosition)

	// trace phase - mark as start of phase
	startPhaseSpan(chain.ExecutionPosition, chandler.id)

	// Execute all function
	for _, function := range phase.GetFunctions() {

		switch {
		// If function
		case function.GetName() != "":
			log.Printf("[request `%s`] Executing function `%s`",
				chandler.id, function.GetName())
			if result == nil {
				result, err = executeFunction(chain, function, request)
			} else {
				result, err = executeFunction(chain, function, result)
			}
			if err != nil {
				stopPhaseSpan(chain.ExecutionPosition)
				return nil, err
			}
		// If callback
		case function.CallbackUrl != "":
			log.Printf("[request `%s`] Executing callback `%s`",
				chandler.id, function.CallbackUrl)
			if result == nil {
				err = executeCallback(chain, function, request)
			} else {
				err = executeCallback(chain, function, result)
			}
			if err != nil {
				stopPhaseSpan(chain.ExecutionPosition)
				return nil, err
			}

		// If modifier
		default:
			log.Printf("[request `%s`] Executing modifier", chandler.id)
			if result == nil {
				result, err = function.Mod(request)
			} else {
				result, err = function.Mod(result)
			}
			if err != nil {
				err = fmt.Errorf("Phase(%d), error: Failed at modifier, %v",
					chain.ExecutionPosition, err)
				return nil, err
			}
			if result == nil {
				result = []byte("")
			}
		}
	}

	log.Printf("[request `%s`] Phase %d completed successfully", chandler.id, chain.ExecutionPosition)

	// trace phase - mark as end of phase
	stopPhaseSpan(chain.ExecutionPosition)

	// Update execution position
	chain.UpdateExecutionPosition()

	return result, nil
}

// forwardAsync forward async request to faaschain
func forwardAsync(chandler *chainHandler, result []byte) ([]byte, error) {

	// get chain
	chain := chandler.getChain()

	// Get chain state
	chainState := chain.GetState()

	// Build request
	uprequest := buildRequest(chandler.id,
		string(chainState),
		chandler.query,
		result,
		chandler.stateM.state)

	// Make request data
	data, _ := uprequest.encode()

	// build url for calling the chain in async
	httpreq, _ := http.NewRequest(http.MethodPost, chandler.asyncUrl, bytes.NewReader(data))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")

	// extend req span for async call
	extendReqSpan(chain.ExecutionPosition-1, chandler.asyncUrl, httpreq)

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

// handleResponse Handle request Response for a faaschain perform response/asyncforward
func handleResponse(chandler *chainHandler, result []byte) ([]byte, error) {

	// get chain
	chain := chandler.getChain()

	switch {

	// If chain has only one phase or if the chain has completed excution return
	case chain.CountPhases() == 1 || chain.GetCurrentPhase() == nil:
		chandler.finished = true
		return result, nil

	// In default case we forward the partial chain to same chain-function
	default:
		// forward the chain request
		resp, forwardErr := forwardAsync(chandler, result)
		if forwardErr != nil {
			return nil, fmt.Errorf("Phase(%d): error: %v, %s, url %s",
				chain.ExecutionPosition, forwardErr,
				string(resp), chandler.asyncUrl)
		}

		log.Printf("[Request `%s`] Async request submitted for Phase %d",
			chandler.id, chain.ExecutionPosition)
	}

	return []byte(""), nil
}

// handleFailure handles failure with failure handler and call finally
func handleFailure(chandler *chainHandler, context *faaschain.Context, err error) {
	context.State = faaschain.StateFailure
	// call failure handler if available
	if chandler.getChain().FailureHandler != nil {
		log.Printf("[request `%s`] Calling failure handler for error, %v",
			chandler.id, err)
		chandler.getChain().FailureHandler(err)
	}
	// call finally handler if available
	if chandler.getChain().Finally != nil {
		log.Printf("[request `%s`] Calling Finally handler with state: %s",
			chandler.id, faaschain.StateFailure)
		chandler.getChain().Finally(faaschain.StateFailure)
	}
	log.Fatalf("[request `%s`] Failed, %v", chandler.id, err)
}

// handleChain handle the chain
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
		chandler, data := buildChain(data)

		// MAKE CONTEXT: make the request context from chain
		context := createContext(chandler)

		// DEFINE: Get Chain definition from user implemented Define()
		err := function.Define(chandler.fchain, context)
		if err != nil {
			context.State = faaschain.StateFailure
			log.Fatalf("[request `%s`] Failed to define chain, %v", chandler.id, err)
		}

		// EXECUTE: execute the chain based on current phase
		result, err := execute(chandler, data)
		if err != nil {
			handleFailure(chandler, context, err)
		}

		// HANDLE: Handle the execution state of last phase
		resp, err = handleResponse(chandler, result)
		if err != nil {
			handleFailure(chandler, context, err)
		}

		if chandler.finished {
			log.Printf("[request `%s`] Completed successfully", chandler.id)

			context.State = faaschain.StateSuccess
			if chandler.getChain().Finally != nil {
				log.Printf("[request `%s`] Calling Finally handler with state: %s",
					chandler.id, faaschain.StateSuccess)
				chandler.getChain().Finally(faaschain.StateSuccess)
			}
		}
	}

	// stop req span if request has finished
	stopReqSpan()

	// flash any pending trace item if tracing enabled
	flushTracer()

	return string(resp)
}
