# Faas-flow - Function Composition for Openfaas
[![Build Status](https://travis-ci.org/s8sg/faas-flow.svg?branch=master)](https://travis-ci.org/s8sg/faas-flow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/s8sg/faas-flow?status.svg)](https://godoc.org/github.com/s8sg/faas-flow)
[![OpenTracing Badge](https://img.shields.io/badge/OpenTracing-enabled-blue.svg)](http://opentracing.io)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)
     
> - [x] **Pure**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; **`FaaS`** with **`openfaas`** 
> - [x] **Fast**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;  build with **`go`**    
> - [x] **Secured**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`HMAC`**
> - [x] **Stateless** &nbsp;&nbsp;&nbsp;&nbsp;  by **`design`**   
> - [x] **Tracing**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; with **`open-tracing`**    
> - [x] **Available** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp; as **`faas-flow`** template 

**FYI**: Faasflow is into conceptual state and API which may change and under active development
   
**Dashboard:** Faasflow comes with a dashboard for visualizing dag generated from flow functions.   
Available as https://github.com/s8sg/faas-flow-tower

## Overview

faas-flow allows you to realize OpenFaaS function composition with ease. By defining a simple pipeline, you can orchestrate multiple functions without having to worry about internals.

```go
import faasflow "github.com/s8sg/faas-flow"

func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
  flow.SyncNode().Apply("yourFunc1").Apply("yourFunc2")
}
```
After building and deploying, it will give you a function that orchestrates calling `yourFunc2` with the output of `yourFunc1`

     
## Pipeline Definition
By supplying a number of pipeline operators, complex compostion can be achieved with little work:
![alt overview](https://github.com/s8sg/faas-flow/blob/master/doc/overview.jpg)

The above pipelines can be achieved with little, but powerfull code:
> SYNC Chain
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     flow.SyncNode().Apply("func1").Apply("func2").
	  Modify(func(data []byte) ([]byte, error) {
	  	// Do something
                return data, nil
	  })
     return
}
```
> ASYNC Chain
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     dag := flow.Dag()
     dag.Node("n1").Apply("func1")
     dag.Node("n2").Apply("func2").
        Modify(func(data []byte) ([]byte, error) {
	        // Do something
               	return data, nil
        })
     dag.Node("n3").Callback("storage.io/bucket?id=3345612358265349126&file=result.dat")
     dag.Edge("n1", "n2")
     dag.Edge("n2", "n3")
     return
}
```
> PARALLEL Branching
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     dag := flow.Dag()
     dag.Node("n1").Modify(func(data []byte) ([]byte, error) {
            // do something
            return data, nil
     })
     dag.Node("n2").Apply("func1")
     dag.Node("n3").Apply("func2").Modify(func(data []byte) ([]byte, error) {
            // do something
            return data, nil
     })
     dag.Node("n4").Callback("storage.io/bucket?id=3345612358265349126&file=result")
     dag.Edge("n1", "n2")
     dag.Edge("n1", "n3")
     dag.Edge("n2", "n4")
     dag.Edge("n3", "n4")
     return
}
```
> DYNAMIC Branching  
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
     dag := flow.Dag()
     dag.Node("n1").Modify(func(data []byte) ([]byte, error) {
            return data, nil
     })
     conditionalDags := dag.ConditionalBranch("C", 
                []string{"c1", "c2"}, // possible conditions
		func(response []byte) []string {
                        // for each returned condition the corresponding branch will execute
                        // this function executes in the runtime of condition C
                        return []string{"c1", "c2"}
                },
     )
     conditionalDags["c2"].Node("n1").Apply("func").Modify(func(data []byte) ([]byte, error) {
		return data, nil
     })
     foreachDag := conditionalDags["c1"].ForEachBranch("F",
		func(data []byte) map[string][]byte {
			// for each returned key in the hashmap a new branch will be executed
                        // this function executes in the runtime of foreach F
                        return map[string][]byte{ "f1": data, "f2": data }
                },
     )
     foreachDag.Node("n1").Modify(func(data []byte) ([]byte, error) {
                return data, nil
     }) 
     dag.Node("n2").Callback("storage.io/bucket?id=3345612358265349126&file=result")
     dag.Edge("n1", "C")
     dag.Edge("C", "n2")
     return
}
``` 
## Faas-flow Design
Faas-flow design is not fixed and like any good design its evolving. The current design consideration are made based on the below goals  
> 1. Leverage the openfaas platform   
> 2. Not to violate the notions of function   
> 3. Provide flexibility, scalability and adaptibility   

#### Just as function as any other
Faas-flow is deployed and provisioned just like any other openfaas function. It allows faas-flow to take advantage of rich functionalities available on Openfaas. `faas-flow` provide a openfaas template and just like any other openfaas function it can be deployed with `faas-cli`   
![alt its a function](https://github.com/s8sg/faas-flow/blob/master/doc/design/complete-faas.jpg)

#### Adapter pattern for zero intrumenttaion in code
Faas-flow function follow the adapter pattern. Here the adaptee is the functions and the adapter is `faas-flow` itself. For each **node** execution, `faas-flow` handle the calls to functions. Once the execution is over, it forwards the call to itself. This way the arrangement logic is seperated from functions and implemented in the adapter. As no code instrumentation is needed,  functions becomes completly independent of the composition logic
![alt function is independent of composition](https://github.com/s8sg/faas-flow/blob/master/doc/design/adapter-pattern.jpg)

#### Aggregate pattern as chaining
Aggregatation of seperate function calls are done as chaining. Multiple functions can be called from a single node with order maintained as per the chain. This way one execution node is capable to be implemented as an aggregator function that invokes multiple functions, collects the results, optionally applies business logic, and returns a consolidated response to the client. Faas-flow fuses the adapter pattern and aggregate pattern to achive more complex usecases
![alt aggregation](https://github.com/s8sg/faas-flow/blob/master/doc/design/aggregate-pattern.jpg)

#### Event driven iteration
Openfaas uses [Nats](https://nats.io) for event delivery and faas-flow uses openfaas platform without depending on other external system. Node execution in `faas-flow` starts by a completion event of one or more previous nodes. A completion event denotes that all the previous dependent nodes has completed. Event carries the execution state, which is used to identify the next node to execute. With events faas-flow asynchronously carry-on execution of nodes by iterating itself over and over till all nodes has been executed
![alt aggregation](https://github.com/s8sg/faas-flow/blob/master/doc/design/event-driven-iteration.jpg)

#### 3rd party KV store for coordination 
When executing branches, one node is dependent of multiple predecessor nodes. In that scenario the event for completion is genearted by coordination of earlier nodes. Like any distributed system the coordination are achived via a centralized service. Faas-flow keeps the logic of the coordination controller inside of faas-flow function and let user use any external 3rd party solution. User can implement `StateStore` and use any synchronous KV stores 
![alt coordination](https://github.com/s8sg/faas-flow/blob/master/doc/design/3rd-party-statestore.jpg)

#### 3rd party Storage for intermediate data
Results from function execution and intermidiate data can be handled by the user manually. Although `faas-flow` provides state-store for intermediate result storage which automatically initialize, store, retrive and remove data between nodes. This fits great for data pipelines. Faas-flow keeps the logic of storage controller inside of faas-flow function and let user use any external 3rd party object storage. User can implement `DataStore` and use any object stores 
![alt coordination](https://github.com/s8sg/faas-flow/blob/master/doc/design/3rd-party-storage.jpg)


## Getting Started
This example implements a flow to `Greet`
    
#### Get the `faas-flow` template with `faas-cli`
```
faas template pull https://github.com/s8sg/faas-flow
```
   
#### Create a new `func` with `faas-flow` template 
```bash
faas new greet --lang faas-flow
```
   
#### Edit `greet.yml`
Edit function stack file `greet.yml`
```yaml
  greet:
    lang: faas-flow
    handler: ./greet
    image: greet:latest
    environment:
      read_timeout: 120 # A value larger than `max` of all execution times of Nodes
      write_timeout: 120 # A value larger than `max` of all execution times of Nodes
      write_debug: true
      combine_output: false
    environment_file:
      - flow.yml
``` 
     
#### Add `flow.yml`
Add a seperate file `flow.yml` with faas-flow related configuration.
```yaml
environment:
  workflow_name: "greet" # The name of the flow function, faasflow use this to forward completion event
  gateway: "gateway:8080" # The address of openfaas gateway, faasflow use this to forward completion event
  # gateway: "gateway.openfaas:8080" # For K8 
  enable_tracing: false # tracing allow to trace internal node execution with opentracing
  enable_hmac: false # hmac adds extra layer of security by validating the event source
```
      
#### Edit Defnition 
Edit `greet/handler.go` and Update `Define()`
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
      flow.SyncNode().
	  Modify(func(data []byte) ([]byte, error) {
	  	result := "Hello " + string(data)
		return []byte(result), nil
	  })
      return nil
}
```

     
#### Build and Deploy
Build and deploy
```bash
faas build
faas deploy
```
> This function will generate one Synchronous node as:     
> ```
> Modify("name") -> Hello name
> ```
All calls will be performed in one single execution of the function, and result will be returned to the callee
     
#### Invoke
```
echo "Adam" | faas invoke greet
```
          
## Request Tracking by ID
Request can be traced from the log by `RequestId`. For each new Request a unique `RequestId` is generated. 
```bash
2018/08/13 07:51:59 [request `bdojh7oi7u6bl8te4r0g`] Created
2018/08/13 07:52:03 [Request `bdojh7oi7u6bl8te4r0g`] Received
```
    
## Request Tracing by Open-Tracing 
    
Request tracing can be enabled by providing the `trace_server` and enabling tracing.  
Edit `flow.yml` as:
```yaml
      enable_tracing: true
      trace_server: "jaegertracing:5775"
``` 
    
### Start The Trace Server 
`jaeger` (opentracing-1.x) used for traceing   
Quick start with jaegertracing: https://www.jaegertracing.io/docs/1.8/getting-started/   
    
Below is an example of tracing for: https://github.com/s8sg/branching-in-faas-flow    
      
![alt multi node](https://github.com/s8sg/faas-flow/blob/master/doc/tracing.png)
    
     
    
## Using context
Context provide verious function such as:   
  **DataStore** to store data,    
  **HttpQuery** to retrivbe original request queries,   
  **State*** to get flow state,  
  **Node** to get current node 
etc.  

### Manage Data Accross Node with `DataStore`  
Faas-flow uses the `DataStore` to store partially completed data and request context data. In faas-flow any dag that forwards data between two nodes need `DataStore`.    
faas-flow allow user to define custom datastore with `DataStore` interface.   
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
Once a `DataStore` is set `faas-flow` uses the same to store intermidiate result inbetween nodes   
    
Context uses `DataStore` to store/retrive data. User can do the same by 
calling `Get()` and `Set()` from `context`:
```go
     flow.SyncNode().
     Modify(func(data []byte) {
	  // parse data and set to be used later
          // json.Unmarshal(&req, data)
          context.Set("commitsha", req.Sha)
     }).
     Apply("myfunc").
     Modify(func(data []byte) {
          // retrived the data that was set in the context
          commitsha, _ = context.GetString("commitsha")
          // use the query
     })
```
* **[MinioDataStore](https://github.com/s8sg/faas-flow-minio-datastore)** allows to store data in **amazon s3** or local **minio DB**


### Manage State of Pipeline in a DAG with `StateStore`
Any DAG which has a branch needs external statestore which can be a 3rd party **Synchoronous KV store**. `Faas-flow` uses `StateStore` top maintain state of Node execution in a dag which has branches.   
Faas-flow allows user to define custom statestore with `StateStore` interface.   
```go
type StateStore interface {
        // Configure the StateStore with flow name and request ID
        Configure(flowName string, requestId string)
        // Initialize the StateStore (called only once in a request span)
        Init() error
        // Set a value (override existing, or create one)
        Set(key string, value string) error
        // Get a value
        Get(key string) (string, error)
        // Compare and Update a value
        Update(key string, oldValue string, newValue string) error
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
  
* **[ConsulStateStore](https://github.com/s8sg/faas-flow-consul-statestore)** statestore implementation with **consul**    
* **[EtcdStateStore](https://github.com/s8sg/faas-flow-etcd-statestore)** statewtore implementation with **etcd**      


### Geting Http Query to Workflow: 
Http Query to flow can be used from context as
```go
    flow.SyncNode().Apply("myfunc", 
         Query("auth-token", context.Query.Get("token"))). // pass as a function query
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
     // Define flow
     flow.SyncNode().Modify(func(data []byte) {
	  // parse data and set to be used later
          // json.Unmarshal(&req, data)
          context.Set("commitsha", req.Sha)
     }).
     Apply("myfunc").Modify(func(data []byte) {
          // retrived the data in different node from context
          commitsha, _ = context.GetString("commitsha")
     })
     flow.OnFailure(func(err error) {
          // failure handler
     }) 
     flow.Finally(func() {
          // delete the state resource
          context.Del("commitsha")
     })
}
```
 
## Contribute:
> **Issue/Suggestion** Create an issue at [faas-flow-issue](https://github.com/s8sg/faas-flow/issues).  
> **ReviewPR/Implement** Create Pull Request at [faas-flow-pr](https://github.com/s8sg/faas-flow/issues).  
