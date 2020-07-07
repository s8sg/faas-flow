package main

import (
	"github.com/faasflow/runtime/controller"
	"handler/config"
	"handler/openfaas"
	"log"
)

func main() {
	runtime := &openfaas.OpenFaasRuntime{}
	port := 8082
	readTimeout := config.ReadTimeout()
	writeTimeout := config.WriteTimeout()
	log.Fatal(controller.StartServer(runtime, port, readTimeout, writeTimeout))
}
