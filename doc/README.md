## faasflow doc

Faasflow guide helps you to get started with faasflow pipeline with example

### Basic
#### Function are workflow agnostic
Faasflow makes the openfaas function in your work flow usecase agnostic. 
Faasflow allows you to get rid of glue code in your function and make each function compleltly unaware about the next function. 
It allows one function to be part of mutiple workflows.  
#### Workflow is function
Workflow can be executes just like a function. Faasflow work as a function which cuts the delay for call to one functions and make it take all the advantage of openfaas platform.
Its highly `Scalable`, `Stateless`, `Secured` and `Fast`.


### Agenga

#### Prerequisite
* [Get familier with openfaas](https://docs.openfaas.com/#get-started-with-openfaas)
    
     
**[Hello World](./guide-helloworld)**
* How to create a faasflow
* How to implement the first `helloworld` flow
* What is modifier
* How to execute a flow
   
     
**[Execute functions](./guide-executesync)**
* How to stitch multiple functions in a flow pipeline
* How to track a request with request Id in log
    
      
**[Using Modifier](./guide-modifier)**
* How use modifier to update or validate result
* How to use modifier to update input for next function
* How to use modifier to validate intermediate result
    
    
**[Using Function Response and Failure Handler](./guide-handler)**
* How to use Function response handler instead of modifier
* How to use Function failure handler     
   
      
**[Async Function](./guide-executeasync)**
* How to use async func
* Where sync and async function can be used together 
* How to report result with async function
    
    
**[Callback](./guide-callback)**
* How to use callback with async function
* How to callback on completion or partial request data
   
   
**[Using failure handler with Finally](./guide-failurehandling)**
* How to handle pipeline failure with failure handler
* How to use finally to perform on completion job
   
   
**[Using Context](./guide-context)**
* How to use context to save and retrive value
* How to use context to get query params
* How to use context to get function name and state
   
   
**[State Manager](./guide-statemanager)**
* What is statemanager and default statemanager
* How to implement and manage state with 3rd party statemanager (minioDb)
  
  
**[Hmac for security](./guide-hamc)**
* How HMAC can be used to secure communication
   
    
**[Tracing with Opentracing](./guide-opentracing)**
* How faasflow can be traced with opentracing
