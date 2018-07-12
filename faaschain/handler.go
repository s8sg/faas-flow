package function

import (
	"bytes"
	"fmt"
	"github.com/s8sg/faaschain/sdk"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

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

// Execute function for a phase
func execute(request *sdk.Request) ([]byte, error) {
	var result []byte
	var httpreq *http.Request

	chain, err := request.GetChain()
	if err != nil {
		err = fmt.Errorf("Phase(%d) : %v", chain.ExecutionPosition, err)
		return nil, err
	}

	phase := chain.GetCurrentPhase()

	// Execute all function
	for _, function := range phase.GetFunctions() {
		name := function.GetName()
		params := function.GetParams()
		headers := function.GetHeaders()

		// Check if intermidiate data
		if result == nil {
			httpreq = buildUpstreamRequest(name, request.GetData(), params, headers)
		} else {
			httpreq = buildUpstreamRequest(name, result, params, headers)
		}
		client := &http.Client{}
		resp, err := client.Do(httpreq)
		if err != nil {
			err = fmt.Errorf("Phase(%d) Function(%s) : %v", chain.ExecutionPosition, name, err)
			return nil, err
		}
		result, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("Phase(%d) Function(%s) : %v", chain.ExecutionPosition, name, err)
			return nil, err
		}
	}
	chain.UpdateExecutionPosition()

	if chain.CountPhases() == 1 {
		return result, nil
	} else {
		// TODO: Forward chain for async handle
		return []byte(""), nil
	}
}

// Handle a serverless request
func Handle(req []byte) string {
	request, err := sdk.DecodeRequest(req)
	if err != nil {
		log.Fatalf("failed to parse request object, error %v", err)
	}

	data, err := execute(request)
	if err != nil {
		log.Fatalf("Error(%s): %s", request.GetID(), err)
	}
	return string(data)
}
