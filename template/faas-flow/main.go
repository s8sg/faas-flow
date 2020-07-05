package main

import (
	"handler/config"
	"handler/openfaas"
	"handler/runtime/controller"
	"log"
)

func main() {
	runtime := &openfaas.OpenFaasRuntime{}
	port := 8082
	readTimeout := config.ReadTimeout()
	writeTimeout := config.WriteTimeout()
	log.Fatal(controller.StartServer(runtime, port, readTimeout, writeTimeout))
}
