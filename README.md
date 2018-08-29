# FaaSChain - FaaS pipeline as a function
[![Build Status](https://travis-ci.org/s8sg/faaschain.svg?branch=master)](https://travis-ci.org/s8sg/faaschain)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/s8sg/faaschain?status.svg)](https://godoc.org/github.com/s8sg/faaschain)
[![OpenTracing Badge](https://img.shields.io/badge/OpenTracing-enabled-blue.svg)](http://opentracing.io)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)

> - [x] **Pure**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; **`FaaS`** with **`openfaas`** 
> - [x] **Fast**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;  build with **`go`**    
> - [x] **Secured**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`HMAC`**
> - [x] **Stateless** &nbsp;&nbsp;&nbsp;&nbsp;  by **`design`** (with optional 3rd party integration)   
> - [x] **Tracing**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`open-tracing`**    
> - [x] **Available** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp; as **`faaschain`** template 


     
## Overview
FaaSChain allow you to define your faas functions pipeline and deploy it as a function
![alt overview](https://github.com/s8sg/faaschain/blob/master/doc/overview.jpg)
     
## Pipeline Definition
Create pipeline with simple call
```go
func Define(chain *fchain.Fchain, context *fchain.Context) (err error) {

     // use any 3rd party to maintain state
     context.SetStateManaget(myMinioStatemanager)

     chain.ApplyModifier(func(data []byte) ([]byte, error) {
               // Set value in context with StateManager
               context.Set("raw-image", data)
               return data
        }).
        Apply("facedetect", Header("method","post")).
        ApplyModifier(func(data []byte) ([]byte, error) {
               // perform check
               // ...
               // and replay data
               data, _ :=context.Get("raw-image")
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

FaaSChain supports sync and async function call. By default all call are async. To call a function in Sync, faaschain provide option `faaschain.Sync`:
```
 chain.Apply("function", faaschain.Sync)
```

**One or more `Async` function call results a chain to have multiple phases**
![alt single phase](https://github.com/s8sg/faaschain/blob/master/doc/asynccall.jpg)

**If all calls are `Sync`, chain will have one phase and return the result to the caller**
![alt multi phase](https://github.com/s8sg/faaschain/blob/master/doc/synccall.jpg)
   
    
| Acronyms |  description |
| ---- | ----- |
| Pipeline Definition | User define the chain as a pipeline by implementing the template `Handle()`. For a given chain the definition is always the same |
| Function | A FaaS Function. A function is applied to chain by calling `chain.Apply(funcName, Sync)` or `chain.Apply(funcName)`. By Default function call are async |
| Modifier | A inline function. A inline modifier function is applied as ```chain.ApplyModifier(func(data []byte) ([]byte, error) { return data, nil } )``` |
| Callback | A URL that will be called with the final/partial result. `chain.Callback(url)` |
| Handler | A Failure handler registered as `chain.OnFailure(func(err error){})`. If registered it is called if an error occured | 
| Finally | A Cleanup handler registered as `chain.Finally(func(){})`. If registered it is called at the end if state is `StateFailure` otherwise `StateSuccess` |
| Phase | Segment of a pipeline definiton which consist of one or more call to `Function` in Sync, `Modifier` or `Callback`. A pipeline definition has one or more phases. Async call `Apply()` results in a new phase. |
| Context | Request context has the state of request. It abstracts the `StateHandler` and provide API to manage state of the request. Interface `StateHandler{}` can be set by user to use 3rd party storage to manage state. |

## Internal

FaaSChain runs four mejor steps to define and run the pipeline
![alt internal](https://github.com/s8sg/faaschain/blob/master/doc/internal.jpg)

| Step |  description |
| ---- | ----- |
| Build Chain | Identify a request and build a chain. A incoming request could be a partially finished pipeline or a fresh raw request. For a partial request `faaschain` parse and understand the state of the pipeline from the incoming request |
| Get Definition |  FaasChain create simple **pipeline-definition** with one or multiple phases based on the chain defined at `Define()` function in `handler.go`. A **pipeline-definition** consist of multiple `phases`. Each `Phase` includes one or more `Function Call`, `Modifier` or `Callback`. Always a single `phase` is executed in a single invokation of the chain. A same chain always outputs to same pipeline definition, which allows `faaschain` to be completly `stateless`|
| Execute | Execute executes a `Phase` by calling the `Modifier`, `Functions` or `Callback` based on how user defines the pipeline. Only one `Phase` gets executed at a single execution of `faaschain function`. |
| Repeat Or Response | If pipeline is not yet completed, FaasChain forwards the remaining pipeline with `partial execution state` and the `partial result` to the same `chain function` via `gateway`. If the pipeline has only one phase or completed `faaschain` returns the output to the gateway otherwise it returns `empty`| 
   
   
## Example
https://github.com/s8sg/faaschain/tree/master/example


## Getting Started
    
#### Get the `faaschain` template with `faas-cli`
```
faas-cli template pull https://github.com/s8sg/faaschain
```
   
#### Create a new `func` with `faaschain` template 
```bash
faas-cli new test-chain --lang faaschain
```
   
#### Edit the `test-chain.yml`
```yaml
  test-chain:
    lang: faaschain
    handler: ./test-chain
    image: test-chain:latest
    environment:
      read_timeout: 120
      write_timeout: 120
      write_debug: true
      combine_output: false
    environment_file:
      - chain.yml
```
> `read_timeout` : A value larger than `max` phase execution time.     
> `write_timeout` : A value larger than `max` phase execution time.     
> `write_debug`: It enables the debug msg in logs.   
> `combine_output` : It allows debug msg to be excluded from `output`.  
     
     
#### Add `chain.yml` with faaschain configuration
To make the stack.yml look clean we can create a seperate `chain.yml` with faaschain related configuration.
```yaml
environment:
  chain_name: "test-chain"
  gateway: "gateway:8080"
  enable_tracing: false
  trace_server: ""
  enable_hmac: false
```

> `chain_name` : The name of the chain function. Faaschain use this to forward partial request.   
> `gateway` : We need to tell faaschain the address of openfaas gateway. All calls are made via gateway
> ```
>              # swarm
>              gateway: "gateway:8080"
>              # k8
>              gateway: "gateway.openfaas:8080"
> ```
> `enable_tracing` : It ebales the opentracing for requests and their phases.  
> `trace_server` : The address of opentracing backend jaeger.  
> `enable_hmac` : Enable hmac to add extra layer of security for partial request forward.

      
##### Edit the `test-chain/handler.go`  
```go
    chain.Apply("yourFunc1", Header("method","post")).
        ApplyModifier(func(data []byte) ([]byte, error) {
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
>     ApplyModifier()    
> Phase 2:    
>     Apply("yourFunc2")   
>     Callback()    
> ```
     
     
##### Build and Deploy the `test-chain`
     
Build
```bash
faas-cli build -f test-chain.yml
```
     
Deploy
```bash
faas-cli deploy -f test-chain.yml
```
     
##### Invoke
```
cat data | faas-cli invoke --async -f test-chain.yml test-chain
Function submitted asynchronously.
```
          
#### Convert with Sync function
> Edit the `test-chain/handler.go`
```go
    chain.Apply("yourFunc1", Header("method","post"), faaschain.Sync).
        ApplyModifier(func(data []byte) ([]byte, error) {
                // Check, update/customize data, replay data ...   
                return []byte(fmt.Sprintf("{ \"data\" : \"%s\" }", string(data))), nil                
        }).Apply("yourFunc2", Header("method", "post"), faaschain.Sync)
```
> This function will generate one phase as:     
> ```
> Phase 1 :    
>     Apply("yourFunc1")    
>     ApplyModifier()    
>     Apply("yourFunc2")  
> ```
          
##### Invoke (Sync)
```
cat data | faas-cli invoke -f test-chain.yml test-chain > updated_data
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
      
![alt multi phase](https://github.com/s8sg/faaschain/blob/master/doc/tracing.png)
    
     
     
## State Management
The main state in faaschain is the **`execution-position` (next-phase)** and the **`partially`** completed data.    
Apart from that faaschain allow user to define state with `StateManager` interface.   
```go
 type StateManager interface {
	Set(key string, value interface{}) error
	Get(key string) (interface{}, error)
	Del(key string) error
 }
```
    
State manager can be implemented and set by user with request context in faaschain `Define()`:
```go
func Define(chain *faaschain.Fchain, context *faaschain.Context) (err error) {
     // initialize my custom StateManager as myStateManager
     context.SetStateManager(myStateManager)
}
```
    
Once a state manager is set it can be used by calling `Get()` and `Set()` from `context`:
```
     chain.ApplyModifier(func(data []byte) {
          // set the query that was passed to the request
          context.Set("query", os.Getenv("Http_Query"))
     }).Apply("myfunc").
     ApplyModifier(func(data []byte) {
          // retrived the query in different phase from context
          query, _ = context.Get("query")
          httpquery, _ =  query.[string]
          // use the query
     })
```
Once `StateManager` is overridden, all call to `Set()`, `Get()` and `del()` will call the provided `StateManager`

## Cleanup with `Finally()`
Finally provides a way to cleanup context and other resources and do post completion work of the pipeline.
A Finally method can be used on chain as:
```go
func Define(chain *faaschain.Fchain, context *faaschain.Context) (err error) {
     // initialize my custom StateManager as myStateManager
     context.SetStateManager(myStateManager)
     
     // Define chain
     chain.ApplyModifier(func(data []byte) {
          // set the query that was passed to the request
          context.Set("query", os.Getenv("Http_Query"))
     }).Apply("myfunc").
     ApplyModifier(func(data []byte) {
          // retrived the query in different phase from context
          query, _ = context.Get("query")
          httpquery, _ =  query.[string]
          // use the query
     }).Finally(func() {
          // delete the state resource
          context.Del("query")
     })
}
```
 
## Contribute:
> **Issue/Suggestion** Create an issue at [faaschain-issue](https://github.com/s8sg/faaschain/issues).  
> **ReviewPR/Implement** Create Pull Request at [faaschain-pr](https://github.com/s8sg/faaschain/issues).  
