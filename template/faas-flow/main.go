package main

import (
	"fmt"
	"handler/function"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	consulStateStore "github.com/s8sg/faas-flow-consul-statestore"
	minioDataStore "github.com/s8sg/faas-flow-minio-datastore"
	sdk "github.com/s8sg/faas-flow/sdk"
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

var (
	stateStore sdk.StateStore
	dataStore  sdk.DataStore
)

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

		openfaasExecutor := &openFaasExecutor{stateStore: stateStore, dataStore: dataStore}

		responseErr := openfaasExecutor.Handle(req, response)

		for k, v := range response.Header {
			w.Header()[k] = v
		}

		if responseErr != nil {
			errorStr := fmt.Sprintf("[ Failed ] %v\n", responseErr)
			fmt.Printf(errorStr)
			w.Write([]byte(errorStr))
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

func initStateStore() (err error) {
	stateStore, err = function.OverrideStateStore()
	if err != nil {
		return err
	}
	if stateStore == nil {

		consulUrl := os.Getenv("consul_url")
		if len(consulUrl) == 0 {
			consulUrl = "consul.faasflow:8500"
		}

		consulDc := os.Getenv("consul_dc")
		if len(consulDc) == 0 {
			consulDc = "dc1"
		}

		stateStore, err = consulStateStore.GetConsulStateStore(consulUrl, consulDc)

		log.Print("Using default state store (consul)")
	}
	return err
}

func initDataStore() (err error) {
	dataStore, err = function.OverrideDataStore()
	if err != nil {
		return err
	}
	if dataStore == nil {

		/*
			minioUrl := os.Getenv("s3_url")
			if len(minioUrl) == 0 {
				minioUrl = "minio.faasflow:9000"
			}

			minioRegion := os.Getenv("s3_region")
			if len(minioRegion) == 0 {
				minioUrl = "us-east-1"
			}

			secretKeyName := os.Getenv("s3_secret_key_name")
			if len(secretKeyName) == 0 {
				secretKeyName = "s3-secret-key"
			}

			accessKeyName := os.Getenv("s3_access_key_name")
			if len(accessKeyName) == 0 {
				accessKeyName = "s3-access-key"
			}

			tlsEnabled := false
			if connection := os.Getenv("s3_tls"); connection == "true" || connection == "1" {
				tlsEnabled = true
			}

			dataStore, err = minioDataStore.Init(minioUrl, minioRegion, secretKeyName, accessKeyName, tlsEnabled)
		*/
		dataStore, err = minioDataStore.InitFromEnv()

		log.Print("Using default data store (minio)")
	}
	return err
}

func main() {
	readTimeout := parseIntOrDurationValue(os.Getenv("read_timeout"), 10*time.Second)
	writeTimeout := parseIntOrDurationValue(os.Getenv("write_timeout"), 10*time.Second)

	var err error

	err = initStateStore()
	if err != nil {
		log.Fatalf("Failed to initialize the StateStore, %v", err)
	}

	err = initDataStore()
	if err != nil {
		log.Fatalf("Failed to initialize the StateStore, %v", err)
	}

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", 8082),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	http.HandleFunc("/", makeRequestHandler())
	log.Fatal(s.ListenAndServe())
}
