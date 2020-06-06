package executor

import (
	"fmt"
	"net/http"
)

// implements faasflow.EventHandler
type openFaasEventHandler struct {
	currentNodeID string        // used to inject current node id in tracer
	tracer        *traceHandler // handle traces with open-tracing
	flowName      string
	header        http.Header
}

func (eh *openFaasEventHandler) Configure(flowName string, requestID string) {
	eh.flowName = flowName
}

func (eh *openFaasEventHandler) Init() error {
	var err error

	// initialize trace server if tracing enabled
	eh.tracer, err = initRequestTracer(eh.flowName)
	if err != nil {
		return fmt.Errorf("failed to init request tracer, error %v", err)
	}
	return nil
}

func (eh *openFaasEventHandler) ReportRequestStart(requestID string) {
	eh.tracer.startReqSpan(requestID)
}

func (eh *openFaasEventHandler) ReportRequestFailure(requestID string, err error) {
	// TODO: add log
	eh.tracer.stopReqSpan()
}

func (eh *openFaasEventHandler) ReportExecutionForward(currentNodeID string, requestID string) {
	eh.currentNodeID = currentNodeID
}

func (eh *openFaasEventHandler) ReportExecutionContinuation(requestID string) {
	eh.tracer.continueReqSpan(requestID, eh.header)
}

func (eh *openFaasEventHandler) ReportRequestEnd(requestID string) {
	eh.tracer.stopReqSpan()
}

func (eh *openFaasEventHandler) ReportNodeStart(nodeID string, requestID string) {
	eh.tracer.startNodeSpan(nodeID, requestID)
}

func (eh *openFaasEventHandler) ReportNodeEnd(nodeID string, requestID string) {
	eh.tracer.stopNodeSpan(nodeID)
}

func (eh *openFaasEventHandler) ReportNodeFailure(nodeID string, requestID string, err error) {
	// TODO: add log
	eh.tracer.stopNodeSpan(nodeID)
}

func (eh *openFaasEventHandler) ReportOperationStart(operationID string, nodeID string, requestID string) {
	eh.tracer.startOperationSpan(nodeID, requestID, operationID)
}

func (eh *openFaasEventHandler) ReportOperationEnd(operationID string, nodeID string, requestID string) {
	eh.tracer.stopOperationSpan(nodeID, operationID)
}

func (eh *openFaasEventHandler) ReportOperationFailure(operationID string, nodeID string, requestID string, err error) {
	// TODO: add log
	eh.tracer.stopOperationSpan(nodeID, operationID)
}

func (eh *openFaasEventHandler) Flush() {
	eh.tracer.flushTracer()
}
