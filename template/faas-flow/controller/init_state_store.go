package controller

import (
	"log"

	"handler/config"
	"handler/function"

	consulStateStore "github.com/s8sg/faas-flow-consul-statestore"
	"github.com/s8sg/faas-flow/sdk"
)

func initStateStore() (stateStore sdk.StateStore, err error) {
	stateStore, err = function.OverrideStateStore()
	if err != nil {
		return nil, err
	}

	if stateStore == nil {
		log.Print("Using default state store (consul)")

		consulURL := config.ConsulURL()
		consulDC := config.ConsulDC()

		stateStore, err = consulStateStore.GetConsulStateStore(consulURL, consulDC)
	}

	return stateStore, err
}
