package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// HttpResponse of function call
type HttpResponse struct {

	// Body the body will be written back
	Body []byte

	// StatusCode needs to be populated with value such as http.StatusOK
	StatusCode int

	// Header is optional and contains any additional headers the function response should set
	Header http.Header
}

// HttpRequest of function call
type HttpRequest struct {
	Body        []byte
	Header      http.Header
	QueryString string
	Method      string
	Host        string
}

// FunctionHandler used for a serverless Go method invocation
type FunctionHandler interface {
	Handle(req *HttpRequest, response *HttpResponse) (err error)
}

func makeRequestHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var input []byte

		if r.Body != nil {
			defer r.Body.Close()

			bodyBytes, bodyErr := ioutil.ReadAll(r.Body)

			if bodyErr != nil {
				fmt.Printf("Error reading body from request.")
			}

			input = bodyBytes
		}

		req := &HttpRequest{
			Body:        input,
			Header:      r.Header,
			Method:      r.Method,
			QueryString: r.URL.RawQuery,
			Host:        r.Host,
		}

		response := &HttpResponse{}
		response.Header = make(map[string][]string)

		openfaasExecutor := &openFaasExecutor{}
		responseErr := openfaasExecutor.Handle(req, response)

		for k, v := range response.Header {
			w.Header()[k] = v
		}

		if responseErr != nil {
			fmt.Printf("[ Failed ] %v\n", responseErr)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if response.StatusCode == 0 {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(response.StatusCode)
			}
		}

		w.Write(response.Body)
	}
}

func parseIntOrDurationValue(val string, fallback time.Duration) time.Duration {
	if len(val) > 0 {
		parsedVal, parseErr := strconv.Atoi(val)
		if parseErr == nil && parsedVal >= 0 {
			return time.Duration(parsedVal) * time.Second
		}
	}

	duration, durationErr := time.ParseDuration(val)
	if durationErr != nil {
		return fallback
	}
	return duration
}

func main() {
	readTimeout := parseIntOrDurationValue(os.Getenv("read_timeout"), 10*time.Second)
	writeTimeout := parseIntOrDurationValue(os.Getenv("write_timeout"), 10*time.Second)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", 8082),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	http.HandleFunc("/", makeRequestHandler())
	log.Fatal(s.ListenAndServe())
}
