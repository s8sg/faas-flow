package controller

import (
	"fmt"
	"log"
	"net/http"

	"handler/config"
)

// StartServer starts the flow function
func StartServer() {
	readTimeout := config.ReadTimeout()
	writeTimeout := config.WriteTimeout()

	var err error

	stateStore, err = initStateStore()
	if err != nil {
		log.Fatalf("Failed to initialize the StateStore, %v", err)
	}

	dataStore, err = initDataStore()
	if err != nil {
		log.Fatalf("Failed to initialize the StateStore, %v", err)
	}

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", 8082),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		Handler:        router(),
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	log.Fatal(s.ListenAndServe())
}
