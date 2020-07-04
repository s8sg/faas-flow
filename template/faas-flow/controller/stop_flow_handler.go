package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/s8sg/faas-flow/sdk/executor"
)

func stopFlowHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Pausing flow %s\n", id)

	flowExecutor := executor.CreateFlowExecutor(ex, nil)
	err := flowExecutor.Stop(id)
	if err != nil {
		return nil, fmt.Errorf("failed to stop request %s, check if request is active", id)
	}

	return []byte("Successfully stopped request " + id), nil
}
