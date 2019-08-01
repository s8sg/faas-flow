package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	faasflow "github.com/s8sg/faas-flow"
)

// buildURL builds openfaas function execution url for the flow
func buildURL(gateway, rpath, function string) string {
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, rpath+"/"+function)
	return u.String()
}

// readSecret reads a secret from /var/openfaas/secrets or from
// env-var 'secret_mount_path' if set.
func readSecret(key string) (string, error) {
	basePath := "/var/openfaas/secrets/"
	if len(os.Getenv("secret_mount_path")) > 0 {
		basePath = os.Getenv("secret_mount_path")
	}

	readPath := path.Join(basePath, key)
	secretBytes, readErr := ioutil.ReadFile(readPath)
	if readErr != nil {
		return "", fmt.Errorf("unable to read secret: %s, error: %s", readPath, readErr)
	}
	val := strings.TrimSpace(string(secretBytes))
	return val, nil
}

// getGateway return the gateway address from env
func getGateway() string {
	gateway := os.Getenv("gateway")
	if gateway == "" {
		gateway = "gateway.openfaas:8080"
	}
	return gateway
}

// getWorkflowNameFromHostFromHost returns the flow name from env
func getWorkflowNameFromHost(host string) string {
	matches := re.FindAllString(host, -1)
	if matches[0] != "" {
		return matches[0]
	}
	return ""
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

// isDagExportRequest check if dag export request
func isDagExportRequest(req *HttpRequest) bool {
	values, err := url.ParseQuery(req.QueryString)
	if err != nil {
		return false
	}

	if strings.ToUpper(values.Get("export-dag")) == "TRUE" {
		return true
	}
	return false
}

// buildHttpRequest build upstream request for function
func buildHttpRequest(url string, method string, data []byte, params map[string][]string,
	headers map[string]string) (*http.Request, error) {

	queryString := makeQueryStringFromParam(params)
	if queryString != "" {
		url = url + queryString
	}

	httpreq, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		httpreq.Header.Add(key, value)
	}

	return httpreq, nil
}

// executeFunction executes a function call
func executeFunction(gateway string, operation *faasflow.FaasOperation, data []byte) ([]byte, error) {
	var err error
	var result []byte

	name := operation.Function
	params := operation.GetParams()
	headers := operation.GetHeaders()

	url := buildURL("http://"+gateway, "function", name)

	method := os.Getenv("default-method")
	if method == "" {
		method = "POST"
	}

	if m, ok := headers["method"]; ok {
		method = m
	}

	httpreq, err := buildHttpRequest(url, method, data, params, headers)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot connect to Function on URL: %s", url)
	}

	if operation.Requesthandler != nil {
		operation.Requesthandler(httpreq)
	}

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()
	if operation.OnResphandler != nil {
		result, err = operation.OnResphandler(resp)
	} else {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			err = fmt.Errorf("invalid return status %d while connecting %s", resp.StatusCode, url)
			result, _ = ioutil.ReadAll(resp.Body)
		} else {
			result, err = ioutil.ReadAll(resp.Body)
		}
	}

	return result, err
}

// executeCallback executes a callback
func executeCallback(operation *faasflow.FaasOperation, data []byte) error {
	var err error

	cburl := operation.CallbackUrl
	params := operation.GetParams()
	headers := operation.GetHeaders()

	method := os.Getenv("default-method")
	if method == "" {
		method = "POST"
	}

	if m, ok := headers["method"]; ok {
		method = m
	}

	httpreq, err := buildHttpRequest(cburl, method, data, params, headers)
	if err != nil {
		return fmt.Errorf("cannot connect to Function on URL: %s", cburl)
	}

	if operation.Requesthandler != nil {
		operation.Requesthandler(httpreq)
	}

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if operation.OnResphandler != nil {
		_, err = operation.OnResphandler(resp)
	} else {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			cbresult, _ := ioutil.ReadAll(resp.Body)
			err := fmt.Errorf("%v:%s", err, string(cbresult))
			return err
		}
	}
	return err

}
