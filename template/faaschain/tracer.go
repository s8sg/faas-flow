package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
	"os"
	"strings"
	"time"
)

var (
	closer     io.Closer
	reqSpan    opentracing.Span
	phaseSpans map[int]opentracing.Span
)

func isTracingEnabled() bool {
	tracing := os.Getenv("enable_tracing")
	if strings.ToUpper(tracing) == "TRUE" {
		return true
	}
	return false
}

func getTraceServer() string {
	traceServer := os.Getenv("trace_server")
	if traceServer == "" {
		traceServer = "jaegertracing:5775"
	}
	return traceServer
}

func initGlobalTracer(chainName string) error {

	if !isTracingEnabled() {
		fmt.Fprintf(os.Stderr, "tracing is disabled\n")
		return nil
	}

	agentPort := getTraceServer()

	fmt.Fprintf(os.Stderr, "tracing is enabled, agent %s\n", agentPort)

	cfg := config.Configuration{
		ServiceName: chainName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  agentPort,
		},
	}

	tracer, traceCloser, err := cfg.NewTracer(
		config.Logger(jaeger.StdLogger),
	)
	if err != nil {
		return fmt.Errorf("Failed to init tracer, error %v", err.Error())
	}

	closer = traceCloser

	opentracing.SetGlobalTracer(tracer)

	phaseSpans = make(map[int]opentracing.Span)

	return nil
}

// TODO: make use of context.Background()
func startReqSpan(reqId string) {
	if !isTracingEnabled() {
		return
	}

	reqSpan = opentracing.GlobalTracer().StartSpan(reqId)
}

func stopReqSpan() {
	if !isTracingEnabled() {
		return
	}

	reqSpan.Finish()
}

// TODO: make use of context.Background()
func startPhaseSpan(phase int, reqId string) {
	if !isTracingEnabled() {
		return
	}

	phasename := fmt.Sprintf("%s-phase-%d", reqId, phase)
	phaseSpan := opentracing.GlobalTracer().StartSpan(
		phasename, opentracing.ChildOf(reqSpan.Context()))
	phaseSpans[phase] = phaseSpan
}

func stopPhaseSpan(phase int) {
	if !isTracingEnabled() {
		return
	}

	phaseSpans[phase].Finish()
}

func flushTracer() {
	if !isTracingEnabled() {
		return
	}

	closer.Close()
}
