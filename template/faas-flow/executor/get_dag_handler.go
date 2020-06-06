package executor

import (
	"fmt"
	"log"
	"net/http"

	"github.com/s8sg/faas-flow/sdk/executor"
	"github.com/s8sg/faas-flow/sdk/exporter"
)

func getDagHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Println("Exporting flow's DAG")

	flowExporter := exporter.CreateFlowExporter(ex)
	resp, err := flowExporter.Export()
	if err != nil {
		return nil, fmt.Errorf("failed to export dag, error %v", err)
	}

	return resp, nil
}
