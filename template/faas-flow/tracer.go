package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	//"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
	"net/http"
	"os"
	"time"
)

type traceHandler struct {
	tracer opentracing.Tracer
	closer io.Closer

	reqSpan    opentracing.Span
	reqSpanCtx opentracing.SpanContext

	nodeSpans map[string]opentracing.Span
}

// CustomHeadersCarrier satisfies both TextMapWriter and TextMapReader
type CustomHeadersCarrier struct {
	envMap map[string]string
}

// buildCustomHeadersCarrier builds a CustomHeadersCarrier from env
func buildCustomHeadersCarrier(header http.Header) *CustomHeadersCarrier {
	carrier := &CustomHeadersCarrier{}
	carrier.envMap = make(map[string]string)

	for k, v := range header {
		if k == "Uber-Trace-Id" && len(v) > 0 {
			key := "uber-trace-id"
			carrier.envMap[key] = v[0]
			break
		}
	}

	return carrier
}

// ForeachKey conforms to the TextMapReader interface
func (c *CustomHeadersCarrier) ForeachKey(handler func(key, val string) error) error {
	for key, value := range c.envMap {
		err := handler(key, value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ForeachKey key %s value %s, error %v",
				key, value, err)
			return err
		}
	}
	return nil
}

// Set conforms to the TextMapWriter interface
func (c *CustomHeadersCarrier) Set(key, val string) {
	c.envMap[key] = val
}

// getTraceServer get the traceserver address
func getTraceServer() string {
	traceServer := os.Getenv("trace_server")
	if traceServer == "" {
		traceServer = "jaegertracing:5775"
	}
	return traceServer
}

// initRequestTracer init global trace with configuration
func initRequestTracer(flowName string) (*traceHandler, error) {
	tracerObj := &traceHandler{}

	agentPort := getTraceServer()

	cfg := config.Configuration{
		ServiceName: flowName,
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

	opentracer, traceCloser, err := cfg.NewTracer(
		config.Logger(jaeger.StdLogger),
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to init tracer, error %v", err.Error())
	}

	tracerObj.closer = traceCloser
	tracerObj.tracer = opentracer
	tracerObj.nodeSpans = make(map[string]opentracing.Span)

	return tracerObj, nil
}

// startReqSpan starts a request span
func (tracerObj *traceHandler) startReqSpan(reqId string) {
	tracerObj.reqSpan = tracerObj.tracer.StartSpan(reqId)
	tracerObj.reqSpan.SetTag("request", reqId)
	tracerObj.reqSpanCtx = tracerObj.reqSpan.Context()
}

// continueReqSpan continue request span
func (tracerObj *traceHandler) continueReqSpan(reqId string, header http.Header) {
	var err error

	tracerObj.reqSpanCtx, err = tracerObj.tracer.Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(header),
	)
	if err != nil {
		fmt.Printf("[Request %s] failed to continue req span for tracing, error %v\n", reqId, err)
		return
	}

	tracerObj.reqSpan = nil
	// TODO: Its not Supported to get span from spanContext as of now
	//       https://github.com/opentracing/specification/issues/81
	//       it will support us to extend the request span for nodes
	//reqSpan = opentracing.SpanFromContext(reqSpanCtx)
}

// extendReqSpan extend req span over a request
// func extendReqSpan(url string, req *http.Request) {
func (tracerObj *traceHandler) extendReqSpan(reqId string, lastNode string, url string, req *http.Request) {
	// TODO: as requestSpan can't be regenerated with the span context we
	//       forward the nodeSpan's SpanContext
	// span := reqSpan
	span := tracerObj.nodeSpans[lastNode]

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "POST")
	err := span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	if err != nil {
		fmt.Printf("[Request %s] failed to extend req span for tracing, error %v\n", reqId, err)
	}
	if req.Header.Get("Uber-Trace-Id") == "" {
		fmt.Printf("[Request %s] failed to extend req span for tracing, error Uber-Trace-Id not set\n",
			reqId)
	}
}

// stopReqSpan terminate a request span
func (tracerObj *traceHandler) stopReqSpan() {
	if tracerObj.reqSpan == nil {
		return
	}

	tracerObj.reqSpan.Finish()
}

// startNodeSpan starts a node span
func (tracerObj *traceHandler) startNodeSpan(node string, reqId string) {
	tracerObj.nodeSpans[node] = tracerObj.tracer.StartSpan(
		node, ext.RPCServerOption(tracerObj.reqSpanCtx))

	/*
		 tracerObj.nodeSpans[node] = tracerObj.tracer.StartSpan(
			node, opentracing.ChildOf(reqSpan.Context()))
	*/

	tracerObj.nodeSpans[node].SetTag("async", "true")
	tracerObj.nodeSpans[node].SetTag("request", reqId)
	tracerObj.nodeSpans[node].SetTag("node", node)
}

// stopNodeSpan terminates a node span
func (tracerObj *traceHandler) stopNodeSpan(node string) {
	tracerObj.nodeSpans[node].Finish()
}

// flushTracer flush all pending traces
func (tracerObj *traceHandler) flushTracer() {
	tracerObj.closer.Close()
}
