package main

import (
	"github.com/faasflow/runtime/controller/http"
	"handler/config"
	"handler/openfaas"
	"log"
)

func main() {
	runtime := &openfaas.OpenFaasRuntime{}
	port := 8082
	readTimeout := config.ReadTimeout()
	writeTimeout := config.WriteTimeout()
	log.Fatal(http.StartServer(runtime, port, readTimeout, writeTimeout))
}
