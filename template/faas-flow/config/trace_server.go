package config

import (
	"os"
)

// TraceServer get the traceserver address
func TraceServer() string {
	traceServer := os.Getenv("trace_server")
	if traceServer == "" {
		traceServer = "jaeger.faasflow:5775"
	}
	return traceServer
}
