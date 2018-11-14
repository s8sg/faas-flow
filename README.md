# Faasflow - FaaS pipeline as a function
[![Build Status](https://travis-ci.org/s8sg/faasflow.svg?branch=master)](https://travis-ci.org/s8sg/faasflow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/s8sg/faasflow?status.svg)](https://godoc.org/github.com/s8sg/faasflow)
[![OpenTracing Badge](https://img.shields.io/badge/OpenTracing-enabled-blue.svg)](http://opentracing.io)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)
     
> - [x] **Pure**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; **`FaaS`** with **`openfaas`** 
> - [x] **Fast**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;  build with **`go`**    
> - [x] **Secured**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`HMAC`**
> - [x] **Stateless** &nbsp;&nbsp;&nbsp;&nbsp;  by **`design`** (with optional 3rd party integration)   
> - [x] **Tracing**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`open-tracing`**    
> - [x] **Available** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp; as **`faasflow`** template 

**FYI**: Faasflow is into conceptual state and API which may change and under active development

## Overview

faasflow allows you to realize OpenFaaS function composition with ease. By defining a simple pipeline, you can orchestrate multiple functions without having to worry about internals.

```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
  flow.Apply("yourFunc1", faasflow.Sync).
       Apply("yourFunc2", faasflow.Sync)
}
```
After building and deploying, it will give you a function that orchestrates calling `yourFunc2` with the output of `yourFunc1`

     
## Pipeline Definition
By supplying a number of pipeline operators, complex compostion can be achieved with little work:
![alt overview](https://github.com/s8sg/faasflow/blob/master/doc/overview.jpg)

The above pipeline can be achieved with little, but powerfull code:

```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {

     // use any 3rd party to maintain state
     context.SetStateManager(myMinioStatemanager)

     flow.Modify(func(data []byte) ([]byte, error) {
               // Set value in context with StateManager
               context.Set("raw-image", data)
               return data
        }).
        Apply("facedetect", Header("method","post")).
        Modify(func(data []byte) ([]byte, error) {
               // perform check
               // ...
               // and replay data
               data, _ := context.GetBytes("raw-image")
	       // do modification if needed
	       return data
        }).
        Apply("compress", Header("method","post")).
        Apply("colorify", Header("method","post")).
        Callback("storage.io/bucket?id=3345612358265349126").
        OnFailure(func(err error) {
              // failure handler
        }).
        Finally(func(state string) {
	      // success - state.State = StateSuccess
	      // failure - state.State = StateFailure
              // cleanup code
	      context.del("raw-image")
        })
}
```

## Sync or Async

Faasflow supports sync and async function call. By default all call are async. To call a function in Sync, faasflow provide option `faasflow.Sync`:
```
 flow.Apply("function", faasflow.Sync)
```

**One or more `Async` function call results a pipeline to have multiple phases**
![alt single phase](https://github.com/s8sg/faasflow/blob/master/doc/asynccall.jpg)

**If all calls are `Sync`, pipeline will have one phase and return the result to the caller**
![alt multi phase](https://github.com/s8sg/faasflow/blob/master/doc/synccall.jpg)
   
    
| Acronyms |  description |
| ---- | ----- |
| Pipeline Definition | User define the flow as a pipeline by implementing the template `Handle()`. For a given flow the definition is always the same |
| Function | A FaaS Function. A function is applied to flow by calling `flow.Apply(funcName, Sync)` or `flow.Apply(funcName)`. By Default function call are async |
| Modifier | A inline function. A inline modifier function is applied as ```flow.Modify(func(data []byte) ([]byte, error) { return data, nil } )``` |
| Callback | A URL that will be called with the final/partial result. `flow.Callback(url)` |
| Handler | A Failure handler registered as `flow.OnFailure(func(err error){})`. If registered it is called if an error occured | 
| Finally | A Cleanup handler registered as `flow.Finally(func(){})`. If registered it is called at the end if state is `StateFailure` otherwise `StateSuccess` |
| Phase | Segment of a pipeline definiton which consist of one or more call to `Function` in Sync, `Modifier` or `Callback`. A pipeline definition has one or more phases. Async call `Apply()` results in a new phase. |
| Context | Request context has the state of request. It abstracts the `StateHandler` and provide API to manage state of the request. Interface `StateHandler{}` can be set by user to use 3rd party storage to manage state. |

## Internal

Faasflow runs four mejor steps to define and run the pipeline
![alt internal](https://github.com/s8sg/faasflow/blob/master/doc/internal.jpg)

| Step |  description |
| ---- | ----- |
| Build Workflow | Identify a request and build a flow. A incoming request could be a partially finished pipeline or a fresh raw request. For a partial request `faasflow` parse and understand the state of the pipeline from the incoming request |
| Get Definition |  FaasWorkflow create simple **pipeline-definition** with one or multiple phases based on the flow defined at `Define()` function in `handler.go`. A **pipeline-definition** consist of multiple `phases`. Each `Phase` includes one or more `Function Call`, `Modifier` or `Callback`. Always a single `phase` is executed in a single invokation of the flow. A same flow always outputs to same pipeline definition, which allows `faasflow` to be completly `stateless`|
| Execute | Execute executes a `Phase` by calling the `Modifier`, `Functions` or `Callback` based on how user defines the pipeline. Only one `Phase` gets executed at a single execution of `faasflow function`. |
| Repeat Or Response | If pipeline is not yet completed, FaasWorkflow forwards the remaining pipeline with `partial execution state` and the `partial result` to the same `flow function` via `gateway`. If the pipeline has only one phase or completed `faasflow` returns the output to the gateway otherwise it returns `empty`| 
   
   
## Example
https://github.com/s8sg/faasflow/tree/master/example


## Getting Started
    
#### Get the `faasflow` template with `faas-cli`
```
faas-cli template pull https://github.com/s8sg/faasflow
```
   
#### Create a new `func` with `faasflow` template 
```bash
faas-cli new test-flow --lang faasflow
```
   
#### Edit the `test-flow.yml`
```yaml
  test-flow:
    lang: faasflow
    handler: ./test-flow
    image: test-flow:latest
    environment:
      read_timeout: 120
      write_timeout: 120
      write_debug: true
      combine_output: false
    environment_file:
      - flow.yml
```
> `read_timeout` : A value larger than `max` phase execution time.     
> `write_timeout` : A value larger than `max` phase execution time.     
> `write_debug`: It enables the debug msg in logs.   
> `combine_output` : It allows debug msg to be excluded from `output`.  
     
     
#### Add `flow.yml` with faasflow configuration
To make the stack.yml look clean we can create a seperate `flow.yml` with faasflow related configuration.
```yaml
environment:
  workflow_name: "test-flow"
  gateway: "gateway:8080"
  enable_tracing: false
  trace_server: ""
  enable_hmac: false
```

> `workflow_name` : The name of the flow function. Faasflow use this to forward partial request.   
> `gateway` : We need to tell faasflow the address of openfaas gateway. All calls are made via gateway
> ```
>              # swarm
>              gateway: "gateway:8080"
>              # k8
>              gateway: "gateway.openfaas:8080"
> ```
> `enable_tracing` : It ebales the opentracing for requests and their phases.  
> `trace_server` : The address of opentracing backend jaeger.  
> `enable_hmac` : Enable hmac to add extra layer of security for partial request forward.

      
##### Edit the `test-flow/handler.go`  
```go
    flow.Apply("yourFunc1", Header("method","post")).
        Modify(func(data []byte) ([]byte, error) {
                // Check, update/customize data, replay data ...   
                return []byte(fmt.Sprintf("{ \"data\" : \"%s\" }", string(data))), nil
        }).Apply("yourFunc2", Header("method","post")).
        Callback("http://gateway:8080/function/send2slack", 
                 Header("method", "post"), Query("authtoken", os.Getenv(token)))
```
> This function will generate two phases as:     
> ```
> Phase 1 :    
>     Apply("yourFunc1")    
>     Modify()    
> Phase 2:    
>     Apply("yourFunc2")   
>     Callback()    
> ```
     
     
##### Build and Deploy the `test-flow`
     
Build
```bash
faas-cli build -f test-flow.yml
```
     
Deploy
```bash
faas-cli deploy -f test-flow.yml
```
     
##### Invoke
```
cat data | faas-cli invoke --async -f test-flow.yml test-flow
Function submitted asynchronously.
```
          
#### Convert with Sync function
> Edit the `test-flow/handler.go`
```go
    flow.Apply("yourFunc1", Header("method","post"), faasflow.Sync).
        Modify(func(data []byte) ([]byte, error) {
                // Check, update/customize data, replay data ...   
                return []byte(fmt.Sprintf("{ \"data\" : \"%s\" }", string(data))), nil                
        }).Apply("yourFunc2", Header("method", "post"), faasflow.Sync)
```
> This function will generate one phase as:     
> ```
> Phase 1 :    
>     Apply("yourFunc1")    
>     Modify()    
>     Apply("yourFunc2")  
> ```
          
##### Invoke (Sync)
```
cat data | faas-cli invoke -f test-flow.yml test-flow > updated_data
```

## Request Tracking by ID
Request can be tracked from the log by `RequestId`. For each new Request a unique `RequestId` is generated. 
```bash
2018/08/13 07:51:59 [request `bdojh7oi7u6bl8te4r0g`] Created
2018/08/13 07:52:03 [Request `bdojh7oi7u6bl8te4r0g`] Received
```
    
## Request Tracing by Open-Tracing 
    
Request tracing can be enabled by providing by specifying    
```yaml
      enable_tracing: true
      trace_server: "jaegertracing:5775"
``` 
    
### Start The Trace Server 
`jaeger` (opentracing-1.x) used for traceing   
To start the trace server we run `jaegertracing/all-in-one` as a service.  
```bash
docker service rm jaegertracing
docker pull jaegertracing/all-in-one:latest
docker service create --constraint="node.role==manager" --detach=true \
        --network func_functions --name jaegertracing -p 5775:5775/udp -p 16686:16686 \
        jaegertracing/all-in-one:latest
```
    
Below is an example of tracing for an async request with 3 phases    
      
![alt multi phase](https://github.com/s8sg/faasflow/blob/master/doc/tracing.png)
    
     
     
## State Management
The main state in faasflow is the **`execution-position` (next-phase)** and the **`partially`** completed data.    
Apart from that faasflow allow user to define state with `StateManager` interface.   
```go
 type StateManager interface {
        Init(flowName string, requestId string) error
	Set(key string, value string) error
	Get(key string) (string, error)
	Del(key string) error
 }
```
    
State manager can be implemented and set by user with request context in faasflow `Define()`:
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     // initialize my custom StateManager as minioStateManager
     context.SetStateManager(minioStateManager)
}
```
    
Once a state manager is set it can be used by calling `Get()` and `Set()` from `context`:
```
     flow.Modify(func(data []byte) {
	  // parse data and set to be used later
          // json.Unmarshal(&req, data)
          context.Set("commitsha", req.Sha)
     }).Apply("myfunc").
     Modify(func(data []byte) {
          // retrived the data that was set in the context
          commitsha, _ = context.GetString("commitsha")
          // use the query
     })
```
* **[MinioStateManager](https://github.com/s8sg/faasflowMinioStateManager)** allows to store state in **amazon s3** or local **minio DB**

#### Default `requestEmbedStateManager`: 
By default faasflow template use `requestEmbedStateManager` which embed the state data along with the request for the next phase. For bigger values it is recommended to pass it with custom `StateManager`. 
    
     
Once `StateManager` is overridden, all call to `Set()`, `Get()` and `del()` will call the provided `StateManager`
    
#### Geting Http Query to Workflow: 
Http Query to flow can be used from context as
```go
    flow.Apply("myfunc", Query("auth-token", context.Query.Get("token"))). // pass as a function query
     	  Modify(func(data []byte) {
          	token = context.Query.Get("token") // get query inside modifier
     	  })
```  

### Use **StateManager** to store intermediate result
By default **`partially`** completed data gets forwarded along with the async request. When using external `StateManager` it can be saved and retrived from the `StateManager` if the flag `intermediate_storage` is set. Default is `false`
```yaml
   intermediate_storage: true
```
    
## Cleanup with `Finally()`
Finally provides a way to cleanup context and other resources and do post completion work of the pipeline.
A Finally method can be used on flow as:
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     // initialize my custom StateManager as myStateManager
     context.SetStateManager(myStateManager)
     
     // Define flow
     flow.Modify(func(data []byte) {
	  // parse data and set to be used later
          // json.Unmarshal(&req, data)
          context.Set("commitsha", req.Sha)
     }).Apply("myfunc").
     Modify(func(data []byte) {
          // retrived the data in different phase from context
          commitsha, _ = context.GetString("commitsha")
     }).Finally(func() {
          // delete the state resource
          context.Del("commitsha")
     })
}
```
 
## Contribute:
> **Issue/Suggestion** Create an issue at [faasflow-issue](https://github.com/s8sg/faasflow/issues).  
> **ReviewPR/Implement** Create Pull Request at [faasflow-pr](https://github.com/s8sg/faasflow/issues).  
