# Faas-flow - Function Composition for Openfaas
[![Build Status](https://travis-ci.org/s8sg/faas-flow.svg?branch=master)](https://travis-ci.org/s8sg/faas-flow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/s8sg/faas-flow?status.svg)](https://godoc.org/github.com/s8sg/faas-flow)
[![OpenTracing Badge](https://img.shields.io/badge/OpenTracing-enabled-blue.svg)](http://opentracing.io)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)
     
> - [x] **Pure**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; **`FaaS`** with **`openfaas`** 
> - [x] **Fast**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;  build with **`go`**    
> - [x] **Secured**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`HMAC`**
> - [x] **Stateless** &nbsp;&nbsp;&nbsp;&nbsp;  by **`design`** (DAG needs external `StateStore` and `DataStore`)   
> - [x] **Tracing**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`open-tracing`**    
> - [x] **Available** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp; as **`faas-flow`** template 

**FYI**: Faasflow is into conceptual state and API which may change and under active development

## Overview

faas-flow allows you to realize OpenFaaS function composition with ease. By defining a simple pipeline, you can orchestrate multiple functions without having to worry about internals.

```go
import faasflow "github.com/s8sg/faas-flow"

func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
  flow.Apply("yourFunc1", faasflow.Sync).
       Apply("yourFunc2", faasflow.Sync)
}
```
After building and deploying, it will give you a function that orchestrates calling `yourFunc2` with the output of `yourFunc1`

     
## Pipeline Definition
By supplying a number of pipeline operators, complex compostion can be achieved with little work:
![alt overview](https://github.com/s8sg/faas-flow/blob/master/doc/overview.jpg)

The above pipeline can be achieved with little, but powerfull code:
> SYNC-Call
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     flow.Apply("func1", faasflow.Sync).
          Apply("func2", faasflow.Sync).
	  Modify(func(data []byte) ([]byte, error) {
	  	// Do something
		return data, nil
	  })
     return nil
}
```
> AYNC-Call
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {

     flow.
        Apply("func1").
	Apply("func2").
        Modify(func(data []byte) ([]byte, error) {
	        // Do something
               	return data
        }).
        Callback("storage.io/bucket?id=3345612358265349126&file=" + context.Query.Get("filename")).
        OnFailure(func(err error) {
              // failure handler
        }).
        Finally(func(state string) {
              // cleanup code
        })
	
	return nil
}
```
> DAG-Call
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {

     dag := faasflow.CreateDag()
     dag.AddModifier("mod1", func(data []byte) ([]byte, error) {
     		// do something
		return data, nil
     })
     dag.AddFunction("func1", "function_1_name")
     dag.AddFunction("func2", "function_2_name")
     dag.AddModifier("mod2", func(data []byte) ([]byte, error) {
     		// do something
		return data, nil
     })
     // To Serialize multiple input the dag need be defined with a Serializer
     dag.AddVertex("callback", faasflow.Serializer(func(inputs map[string][]byte) ([]byte, error) {
				          mod2Data := inputs["mod2"]
					  func2Data := inputs["func2"]
				          // Serialize input for callback
					  return data, nil
				    }))
     dag.AddCallback("callback", "storage.io/bucket?id=3345612358265349126&file=" + context.Query.Get("filename"))
				    

     dag.AddEdge("mod1", "func1")
     dag.AddEdge("mod1", "func2")
     dag.AddEdge("func1", "mod2")
     dag.AddEdge("func2", "callback")
     dag.AddEdge("mod2", "callback")
     
     flow.ExecuteDag(dag)
     
     return nil
}

func DefineStateStore() (faasflow.StateStore, error) {
        // use consul StateStore
        consulss, err := consulStateStore.GetConsulStateStore()
        return consulss, err
}

func DefineDataStore() (faasflow.DataStore, error) {
        // use minio DataStore
        miniods, err := minioDataStore.InitFromEnv()
        return miniods, err
}
```


## Sync or Async

Faasflow supports sync and async function call. By default all call are async. To call a function in Sync, faas-flow provide option `faasflow.Sync`:
```
 flow.Apply("function", faasflow.Sync)
```


**If all calls are `Sync`, pipeline will have one Node (Vertex) and return the result to the caller**
![alt single node](https://github.com/s8sg/faas-flow/blob/master/doc/synccall.jpg)

**One or more `Async` function call results a pipeline to have multiple Nodes (Vertex) as a `chain`**
![alt multi node](https://github.com/s8sg/faas-flow/blob/master/doc/asynccall.jpg)

**If pipeline is created as a `dag`, the pipeline will have multiple Nodes(Vertex)**
![alt multi node dag](https://github.com/s8sg/faas-flow/blob/master/doc/asyncdag.jpg)
   
    
| Acronyms |  description |
| ---- | ----- |
| Pipeline Definition | User define the flow as a pipeline by implementing the template `Handle()`. For a given flow the definition is always the same |
| Function | A FaaS Function. A function is applied to flow by calling `flow.Apply(funcName, Sync)` or `flow.Apply(funcName)`. By Default function call are async |
| Modifier | A inline function. A inline modifier function is applied as ```flow.Modify(func(data []byte) ([]byte, error) { return data, nil } )``` |
| Callback | A URL that will be called with the final/partial result. `flow.Callback(url)` |
| Handler | A Failure handler registered as `flow.OnFailure(func(err error){})`. If registered it is called if an error occured | 
| Finally | A Cleanup handler registered as `flow.Finally(func(){})`. If registered it is called at the end if state is `StateFailure` otherwise `StateSuccess` |
| Node | A vertex that represent a segment of a pipeline definiton which consist of one or more call to `Operation`. A pipeline definition has one or more nodes. Async call results in a new node in a chain. A dag is a composition of multiple nodes |
| Context | Request context has the state of request. It abstracts the `StateHandler` and provide API to manage state of the request. Interface `StateHandler{}` can be set by user to use 3rd party storage to manage state. |

## Internal

Faasflow runs four major steps to define and run the pipeline
![alt internal](https://github.com/s8sg/faas-flow/blob/master/doc/internal.jpg)

| Step |  description |
| ---- | ----- |
| Build Workflow | Identify a request and build a flow. A incoming request could be a partially finished pipeline or a fresh raw request. For a partial request `faas-flow` parse and understand the state of the pipeline from the incoming request |
| Get Definition |  FaasWorkflow create simple **pipeline-definition** with one or multiple nodes based on the flow defined at `Define()` function in `handler.go`. A **pipeline-definition** consist of multiple `nodes`. Each `Node` includes one or more `Function Call`, `Modifier` or `Callback`. Always a single `node` is executed in a single invokation of the flow. A same flow always outputs to same pipeline definition, which allows `faas-flow` to be completly `stateless`|
| Execute | Execute executes a `Node` by calling the `Modifier`, `Functions` or `Callback` based on how user defines the pipeline. Only one `Node` gets executed at a single execution of `faas-flow function`. |
| Repeat Or Response | If pipeline is not yet completed, FaasWorkflow forwards the remaining pipeline with `partial execution state` and the `partial result` to the same `flow function` via `gateway`. If the pipeline has only one node or completed `faas-flow` returns the output to the gateway otherwise it returns `empty`| 
   
   
## Example
https://github.com/s8sg/faas-flow-examples


## Getting Started
    
#### Get the `faas-flow` template with `faas-cli`
```
faas-cli template pull https://github.com/s8sg/faas-flow
```
   
#### Create a new `func` with `faas-flow` template 
```bash
faas-cli new test-flow --lang faas-flow
```
   
#### Edit the `test-flow.yml`
```yaml
  test-flow:
    lang: faas-flow
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
> `read_timeout` : A value larger than `max` node execution time.     
> `write_timeout` : A value larger than `max` node execution time.     
> `write_debug`: It enables the debug msg in logs.   
> `combine_output` : It allows debug msg to be excluded from `output`.  
     
     
#### Add `flow.yml` with faas-flow configuration
To make the stack.yml look clean we can create a seperate `flow.yml` with faas-flow related configuration.
```yaml
environment:
  workflow_name: "test-flow"
  gateway: "gateway:8080"
  enable_tracing: false
  trace_server: ""
  enable_hmac: false
```

> `workflow_name` : The name of the flow function. Faasflow use this to forward partial request.   
> `gateway` : We need to tell faas-flow the address of openfaas gateway. All calls are made via gateway
> ```
>              # swarm
>              gateway: "gateway:8080"
>              # k8
>              gateway: "gateway.openfaas:8080"
> ```
> `enable_tracing` : It ebales the opentracing for requests and their nodes.  
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
> This function will generate two nodes as:     
> ```
> Node 1 :    
>     Apply("yourFunc1")    
>     Modify()    
> Node 2:    
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
> Edit the ` at `function/handler.go``
```go
    flow.Apply("yourFunc1", Header("method","post"), faasflow.Sync).
        Modify(func(data []byte) ([]byte, error) {
                // Check, update/customize data, replay data ...   
                return []byte(fmt.Sprintf("{ \"data\" : \"%s\" }", string(data))), nil                
        }).Apply("yourFunc2", Header("method", "post"), faasflow.Sync)
```
> This function will generate one node as:     
> ```
> Node 1 :    
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
    
Below is an example of tracing for an async request with 3 Nodes    
      
![alt multi node](https://github.com/s8sg/faas-flow/blob/master/doc/tracing.png)
    
     
    
## Using request context
Request context provide verious function such as:   
  **DataStore** to store data,    
  **HttpQuery** to retrivbe request query,  
  **State*** to get flow state,  
  **Node** to get current node 
etc.  

### Manage Data Accross Node with `DataStore`
The main state in faas-flow chain is the **`execution-position` (next-Node)** and the **`partially`** completed data.    
Apart from that faas-flow allow user to define state with `DataStore` interface.   
```go
 type DataStore interface {
        // Configure the DaraStore with flow name and request ID
        Configure(flowName string, requestId string)
        // Initialize the DataStore (called only once in a request span)
        Init() error
        // Set store a value for key, in failure returns error
        Set(key string, value string) error
        // Get retrives a value by key, if failure returns error
        Get(key string) (string, error)
        // Del delets a value by a key
        Del(key string) error
        // Cleanup all the resorces in DataStore
        Cleanup() error
 }
```
    
Data Store can be implemented and set by user at the `DefineDataStore()` at `function/handler.go`:
```go
// ProvideDataStore provides the override of the default DataStore
func DefineDataStore() (faasflow.DataStore, error) {
        // initialize minio DataStore
        miniods, err := minioDataStore.InitFromEnv()
        return miniods, err
}
```
    
Once a DataStore is set it can be used by calling `Get()` and `Set()` from `context`:
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
* **[MinioDataStore](https://github.com/s8sg/faas-flow-minio-datastore)** allows to store data in **amazon s3** or local **minio DB**

> **Default `requestEmbedDataStore`:**   
> By default faas-flow template use `requestEmbedDataStore` which embed the state data along with the request for the next node. For bigger values it is recommended to pass it with custom `DataStore`. 
    
Once `DataStore` is overridden, all call to `Set()`, `Get()` and `del()` will call the provided `DataStore`

### Use **DataStore** to store intermediate result
By default **`partially`** completed data gets forwarded along with the async request. When using external `DataStore` it can be saved and retrived from the `DataStore` if the flag `intermediate_storage` is set. Default is `false`
```yaml
   intermediate_storage: true
```
Due to **nats** `1mb` storage limitation, async call may fail. In such scenario using `intermediate_storage` is recommended

### Manage State of Pipeline in a DAG with `StateStore`
In a `faas-flow` DAG execution faas-flow state is not only depends on the execution position, as the DAG execution happens on a shared state, a 3rd party **Synchoronous KV store** can be used as a `StateStore`
`StateStore` provides the below interface:
```go
type StateStore interface {
        // Configure the StateStore with flow name and request ID
        Configure(flowName string, requestId string)
        // Initialize the StateStore (called only once in a request span)
        Init() error
        // create Vertexes for request
        // creates a map[<vertexId>]<Indegree Completion Count>
        Create(vertexs []string) error
        // Increment Vertex Indegree Completion
        // synchronously increment map[<vertexId>] Indegree Completion Count by 1 and return updated count
        IncrementCounter(vertex string) (int, error)
        // Set state of pipeline
        SetState(state bool) error
        // Get State of pipeline
        GetState() (bool, error)
        // Cleanup all the resorces in StateStore (called only once in a request span)
        Cleanup() error
}
```
A `StateStore` can be implemented with any KV Store that provides `Synchronization`. The implemented `StateStore` can be set with `DefineStateStore()` at `function/handler.go`:
```go
// DefineStateStore provides the override of the default StateStore
func DefineStateStore() (faasflow.StateStore, error) {
        consulss, err := consulStateStore.GetConsulStateStore(os.Getenv("consul_url"), os.Getenv("consul_dc"))
        return consulss, err
}
```

* **[ConsulStateStore](https://github.com/s8sg/faas-flow-consul-statestore)** manage state in **consul** for dag execution.  
* **[EtcdStateStore](https://github.com/s8sg/faas-flow-etcd-statestore)** manage state in **etcd** for dag execution.      


### Geting Http Query to Workflow: 
Http Query to flow can be used from context as
```go
    flow.Apply("myfunc", Query("auth-token", context.Query.Get("token"))). // pass as a function query
     	  Modify(func(data []byte) {
          	token = context.Query.Get("token") // get query inside modifier
     	  })
```  

### Other from context:
Node, requestId, State is provided by the `context`
```go
   currentNode := context.GetNode()
   requestId := context.GetRequestId()
   state := context.State
```
for more details check `[faas-flow-GoDoc](https://godoc.org/github.com/s8sg/faas-flow)
   
    
     
## Cleanup with `Finally()`
Finally provides a way to cleanup context and other resources and do post completion work of the pipeline.
A Finally method can be used on flow as:
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     // initialize my custom DataStore as myDataStore
     context.SetDataStore(myDataStore)
     
     // Define flow
     flow.Modify(func(data []byte) {
	  // parse data and set to be used later
          // json.Unmarshal(&req, data)
          context.Set("commitsha", req.Sha)
     }).Apply("myfunc").
     Modify(func(data []byte) {
          // retrived the data in different node from context
          commitsha, _ = context.GetString("commitsha")
     }).Finally(func() {
          // delete the state resource
          context.Del("commitsha")
     })
}
```
 
## Contribute:
> **Issue/Suggestion** Create an issue at [faas-flow-issue](https://github.com/s8sg/faas-flow/issues).  
> **ReviewPR/Implement** Create Pull Request at [faas-flow-pr](https://github.com/s8sg/faas-flow/issues).  
