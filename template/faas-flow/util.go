package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
)

// buildURL builds execution url for the flow
func buildURL(gateway, rPath, function string) string {
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, rPath+"/"+function)
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

// getStopRequestId check if stop request and return the requestID
func getStopRequestId(req *HttpRequest) string {
	values, err := url.ParseQuery(req.QueryString)
	if err != nil {
		return ""
	}

	reqId := values.Get("stop-flow")
	return reqId
}

// getPauseRequestId check if pause request and return the requestID
func getPauseRequestId(req *HttpRequest) string {
	values, err := url.ParseQuery(req.QueryString)
	if err != nil {
		return ""
	}

	reqId := values.Get("pause-flow")
	return reqId
}

// getResumeRequestId check if resume request and return the requestID
func getResumeRequestId(req *HttpRequest) string {
	values, err := url.ParseQuery(req.QueryString)
	if err != nil {
		return ""
	}

	reqId := values.Get("resume-flow")
	return reqId
}
