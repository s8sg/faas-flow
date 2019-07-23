# Faas-flow - Function Composition for [Openfaas](https://github.com/openfaas/faas)
[![Build Status](https://travis-ci.org/s8sg/faas-flow.svg?branch=master)](https://travis-ci.org/s8sg/faas-flow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/s8sg/faas-flow?status.svg)](https://godoc.org/github.com/s8sg/faas-flow)
[![OpenTracing Badge](https://img.shields.io/badge/OpenTracing-enabled-blue.svg)](http://opentracing.io)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)
     
> - [x] **Pure**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; FaaS with [Openfaas](https://github.com/openfaas/faas) 
> - [x] **Fast**      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;  Built with `Go`    
> - [x] **Secured**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; With `HMAC`
> - [x] **Stateless** &nbsp;&nbsp;&nbsp;&nbsp; By design   
> - [x] **Tracing**   &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; With `open-tracing`    
> - [x] **Available** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;As `faas-flow` template 
   
[**Faas-flow tower**](https://github.com/s8sg/faas-flow-tower) visualizes and monitors flow function  

## Overview

Faas-flow allows you to realize OpenFaaS function composition with ease. By defining a simple pipeline, you can orchestrate multiple functions without having to worry about internals

```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
    flow.SyncNode().Apply("Func1").Apply("Func2")
    return
}
```
After building and deploying, it will give you a openfaas function that orchestrates calling `Func2` with the output of `Func1`

## Use Cases

Faas-flow as a function composure provides the back-bone for building  complex solutions and promote automation
   
#### Data Processing Pipeline

Faas-flow can orchestrate a pipeline with long and short running function performing ETL jobs without having to orchestrate them manually or maintaining a separate application. Faas-flow ensures the execution order of several functions running in parallel or dynamically and provides rich construct to aggregate results while maintaining the intermediate data.
   
#### Application Orchestration Workflow

Functions are great for isolating certain functionalities of an application. Although one still need to call the functions, write workflow logic, handle parallel processing and retries on failures. Using Faas-flow you can combine multiple openfaas functions with little codes while your workflow will scale up/down automatically to handle the load
    
#### Function Reusability

Fass-flow allows you to write function only focused on solving one problem without having to worry about the next. It makes function loosely coupled from the business logic promoting reusability. You can write the stateless function and use it across multiple applications, where faas-flow maintains the execution state for individual workflow per requests
     
     
## Pipeline Definition
By supplying a number of pipeline operators, complex compostion can be achieved with little work:
![alt overview](https://github.com/s8sg/faas-flow/blob/master/doc/overview.jpg)

The above pipelines can be achieved with little, but powerfull code:
> SYNC Chain
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
        flow.SyncNode().Apply("func1").Apply("func2").
                Modify(func(data []byte) ([]byte, error) {
                        // do something 
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
                        // do something
                        return data, nil
                })
        dag.Node("n3").Callback("http://gateway:8080/function/fake-storage")
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
        dag.Node("n4", faasflow.Aggregator(func(data map[string][]byte) ([]byte, error) {
                // aggregate branch result data["n2"] and data["n3"]
                return []byte(""), nil
        })).Callback("http://gateway:8080/function/fake-storage")

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
                // do something
                return data, nil
        })
        conditionalDags := dag.ConditionalBranch("C",
                []string{"c1", "c2"}, // possible conditions
                func(response []byte) []string {
                        // for each returned condition the corresponding branch will execute
                        // this function executes in the runtime of condition C
                        return []string{"c1", "c2"}
                },
                faasflow.Aggregator(func(data map[string][]byte) ([]byte, error) {
                        // aggregate all dynamic branches results
                        return []byte(""), nil
                }),
        )

        conditionalDags["c2"].Node("n1").Apply("func1").Modify(func(data []byte) ([]byte, error) {
                // do something
                return data, nil
        })
        foreachDag := conditionalDags["c1"].ForEachBranch("F",
                func(data []byte) map[string][]byte {
                        // for each returned key in the hashmap a new branch will be executed
                        // this function executes in the runtime of foreach F
                        return map[string][]byte{"f1": data, "f2": data}
                },
                faasflow.Aggregator(func(data map[string][]byte) ([]byte, error) {
                        // aggregate all dynamic branches results
                        return []byte(""), nil
                }),
        )
        foreachDag.Node("n1").Modify(func(data []byte) ([]byte, error) {
                // do something
                return data, nil
        })
        dag.Node("n2").Callback("http://gateway:8080/function/fake-storage")
        dag.Edge("n1", "C")
        dag.Edge("C", "n2")
}
```
Full implementions of the above examples are available [here](https://github.com/s8sg/faasflow-example)
## Faas-flow Design
The current design consideration are made based on the below goals  
> 1. Leverage the openfaas platform   
> 2. Not to violate the notions of function   
> 3. Provide flexibility, scalability and adaptibility   

#### Just as function as any other
Faas-flow is deployed and provisioned just like any other openfaas function. It allows faas-flow to take advantage of rich functionalities available on Openfaas. `faas-flow` provide a openfaas template and just like any other openfaas function it can be deployed with `faas-cli`   
![alt its a function](https://github.com/s8sg/faas-flow/blob/master/doc/design/complete-faas.jpg)

#### Adapter pattern for zero intrumenttaion in code
Faas-flow function follow the adapter pattern. Here the adaptee is the functions and the adapter is `faas-flow`. For each node execution, `faas-flow` handle the calls to the functions. Once the execution is over, it forwards an event to itself. This way the arrangement logic is seperated from the functions and is implemented in the adapter. Compositions need no code instrumentations, making functions completly independent of the compositions details
![alt function is independent of composition](https://github.com/s8sg/faas-flow/blob/master/doc/design/adapter-pattern.jpg)

#### Aggregate pattern as chaining
Aggregatation of seperate function calls are done as chaining. Multiple functions can be called from a single node with order maintained as per the chain. This way one execution node can be implemented as an aggregator function that invokes multiple functions, collects the results, optionally applies business logic, and returns a consolidated response to the client or forward to next nodes. Faas-flow fuses the adapter pattern and aggregate pattern to support more complex usecases
![alt aggregation](https://github.com/s8sg/faas-flow/blob/master/doc/design/aggregate-pattern.jpg)

#### Event driven iteration
Openfaas uses [Nats](https://nats.io) for event delivery and faas-flow leverages openfaas platform. Node execution in `faas-flow` starts by a completion event of one or more previous nodes. A completion event denotes that all the previous dependent nodes have completed. The event carries the execution state and identifies the next node to execute. With events faas-flow asynchronously carry-on execution of nodes by iterating itself over and over till all nodes are executed
![alt iteration](https://github.com/s8sg/faas-flow/blob/master/doc/design/event-driven-iteration.jpg)

#### 3rd party KV store for coordination 
When executing branches, one node is dependent on more than one predecessor nodes. In that scenario, the event for completion is generated by coordination of earlier nodes. Like any distributed system the coordination is achieved via a centralized service. Faas-flow keeps the logic of the coordination controller inside of faas-flow implementation and lets the user use any external synchronous KV store by implementing [`StateStore`](https://godoc.org/github.com/s8sg/faas-flow#StateStore) 
![alt coordination](https://github.com/s8sg/faas-flow/blob/master/doc/design/3rd-party-statestore.jpg)

#### 3rd party Storage for intermediate data
Results from function execution and intermediate data can be handled by the user manually. Faas-flow provides data-store for intermediate result storage. It automatically initializes, store, retrieve and remove data between nodes. This fits great for data processing applications. Faas-flow keeps the logic of storage controller inside of Faas-flow implementation and lets the user use any external object storage by implementing [`DataStore`](https://godoc.org/github.com/s8sg/faas-flow#DataStore) 
![alt storage](https://github.com/s8sg/faas-flow/blob/master/doc/design/3rd-party-storage.jpg)
    
    
Faas-flow design is not fixed and like any good design it is evolving. Please contribute to make it better.  

## Getting Started
This example implements a very simple flow to `Greet`
    
#### Get template
Pull `faas-flow` template with the `faas-cli`
```
faas template pull https://github.com/s8sg/faas-flow
```
   
#### Create new flow function
Create a new function using `faas-flow` template
```bash
faas new greet --lang faas-flow
```
   
#### Edit stack
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
      workflow_name: "greet" # The name of the flow function, faasflow use this to forward completion event
    environment_file:
      - flow.yml
``` 
     
#### Add configuration
Add a seperate file `flow.yml` with faas-flow related configuration.
```yaml
environment:
  gateway: "gateway:8080" # The address of openfaas gateway, faasflow use this to forward completion event
  # gateway: "gateway.openfaas:8080" # For K8s 
  enable_tracing: false # tracing allow to trace internal node execution with opentracing
  enable_hmac: true # hmac adds extra layer of security by validating the event source
```
      
#### Edit function defnition 
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
> This function will generate one Synchronous node     
> ```
> Modify("name") -> Hello name
> ```
All calls will be performed in one single execution of the flow function and result will be returned to the callee    
> Note: For flow that has more than one nodes, faas-flow doesn't return any response. External storage or `callback` can be used to retrive async result 
     
#### Invoke
```
echo "Adam" | faas invoke greet
```
          
## Request Tracking by ID
For each new request faas-flow generates a unique `Request Id` for the flow. The same Id is used when logging 
```bash
2018/08/13 07:51:59 [Request `bdojh7oi7u6bl8te4r0g`] Created
2018/08/13 07:52:03 [Request `bdojh7oi7u6bl8te4r0g`] Received
```
The assigned requestId is set on the response header `X-Faas-Flow-Reqid` 

    
## Request Tracing by Open-Tracing 
    
Request tracing can be retrived from `trace_server` once enabled. Tracing is the best way to monitor flows and execution status of each nodes for each requests

#### Edit `flow.yml`
Enable tracing and add trace server as:
```yaml
      enable_tracing: true
      trace_server: "jaegertracing:5775"
``` 
    
#### Start The Trace Server 
`jaeger` (opentracing-1.x) used for tracing backend  
Quick start with jaegertracing: https://www.jaegertracing.io/docs/1.8/getting-started/   
    
#### Use [faas-flow-tower](https://github.com/s8sg/faas-flow-tower)
Retrive the requestID from `X-Faas-Flow-Reqid` header of response     

Below is an example of tracing information for [example-branching-in-faas-flow](https://github.com/s8sg/branching-in-faas-flow) in [faas-flow-tower](https://github.com/s8sg/faas-flow-tower)  
![alt monitoring](https://github.com/s8sg/faas-flow-tower/blob/master/doc/monitoring.png)
    
     
    
## Use of context

Context can be used inside definition for differet usecases. Context provide verious information such as:   
  **HttpQuery** to retrivbe original request queries   
  **State** to get flow state  
  **Node** to get current node    
along with that it wraps the **DataStore** to store data    

#### Store data in context with `DataStore`
Context uses `DataStore` to store/retrive data. User can do the same by 
calling `Get()`, `Set()` and `Del()` from `context`:
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

#### Geting Http Query to Workflow: 
Http Query to flow can be used from context as
```go
    flow.SyncNode().Apply("myfunc", Query("auth-token", context.Query.Get("token"))). // pass as a function query
     	 Modify(func(data []byte) {
          	token = context.Query.Get("token") // get query inside modifier
     	 })
```  

#### Other from context:
Node, requestId, State is provided by the `context`
```go
   currentNode := context.GetNode()
   requestId := context.GetRequestId()
   state := context.State
```
for more details check `[faas-flow-GoDoc](https://godoc.org/github.com/s8sg/faas-flow)


## External `StateStore` for coordination controller
Any DAG which has a branch needs coordination for nodes completion events. Faas-flow implements coordination controller which allows user to use any external Synchoronous KV store. User can define custom state-store with `StateStore` interface.   
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
  
#### Available state-stores:  
* **[ConsulStateStore](https://github.com/s8sg/faas-flow-consul-statestore)** statestore implementation with **consul**    
* **[EtcdStateStore](https://github.com/s8sg/faas-flow-etcd-statestore)** statewtore implementation with **etcd**      

## External `DataStore` for storage controller 
Faas-flow uses the `DataStore` to store partially completed data between nodes and request context data. Faas-flow implements storage controller to handle storage that allows user to use any external object store. User can define custom data-store with `DataStore` interface.   
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

#### Available data-stores:  
* **[MinioDataStore](https://github.com/s8sg/faas-flow-minio-datastore)** allows to store data in **amazon s3** or local **minio DB**

     
## Cleanup with `Finally()`
Finally provides an efficient way to perform post execution steps of the flow. If specified `Finally()` invokes in case of both failure and success of the flow. A Finally method can be set as:
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
