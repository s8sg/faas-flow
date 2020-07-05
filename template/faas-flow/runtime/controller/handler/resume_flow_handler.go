package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/faasflow/sdk/executor"
)

func ResumeFlowHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Resuming flow %s\n", id)

	flowExecutor := executor.CreateFlowExecutor(ex, nil)
	err := flowExecutor.Resume(id)
	if err != nil {
		return nil, fmt.Errorf("failed to resume request %s, check if request is active", id)
	}

	return []byte("Successfully resumed request " + id), nil
}
