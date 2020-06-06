package executor

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`(?m)^[^:.]+\s*`)

// buildURL builds execution url for the flow
func buildURL(gateway, rPath, function string) string {
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, rPath, function)
	return u.String()
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
func isDagExportRequest(query string) bool {
	values, err := url.ParseQuery(query)
	if err != nil {
		return false
	}

	if strings.ToUpper(values.Get("export-dag")) == "TRUE" {
		return true
	}
	return false
}

// getStateRequestID check if state request and return the requestID
func getStateRequestID(query string) string {
	values, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}

	return values.Get("state")
}

// getStopRequestID check if stop request and return the requestID
func getStopRequestID(query string) string {
	values, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}

	return values.Get("stop-flow")
}

// getPauseRequestID check if pause request and return the requestID
func getPauseRequestID(query string) string {
	values, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}

	return values.Get("pause-flow")
}

// getResumeRequestID check if resume request and return the requestID
func getResumeRequestID(query string) string {
	values, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}

	return values.Get("resume-flow")
}
