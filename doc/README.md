## faasflow doc 

Faasflow guide helps you to get started with faasflow pipeline with example

### Agenga

#### Prerequisite
* [Get familier with openfaas](https://docs.openfaas.com/#get-started-with-openfaas)


**[Set Up Environment](./env-setup)**  
Guide to setup environment in docker swarm and in kubernets    
     
     
**[Hello World](./guide-helloworld)**  
Learn how to create your first flow and execute it. Get a overview of the building block and operation such as `modify()` 
       
        
**[Execute functions](./guide-chain-singlenode)**   
Learn how to create a node in a dag and perform multiple operations. Get an overview of the other operations, such as `apply()` and `request()`
    
      
**[Using Modifier](./guide-modifier)** (obsolete)
* How use modifier to update or validate result
* How to use modifier to update input for next function
* How to use modifier to validate intermediate result
    
    
**[Using Function Response and Failure Handler](./guide-handler)** (obsolete)
* How to use Function response handler instead of modifier
* How to use Function failure handler     
   
      
**[Async Function](./guide-executeasync)** (obsolete)
* How to use async func
* Where sync and async function can be used together 
* How to report result with async function
    
    
**[Callback](./guide-callback)** (obsolete)
* How to use callback with async function
* How to callback on completion or partial request data
   
   
**[Using failure handler with Finally](./guide-failurehandling)** (obsolete)
* How to handle pipeline failure with failure handler
* How to use finally to perform on completion job
   
   
**[Using Context](./guide-context)** (obsolete)
* How to use context to save and retrive value
* How to use context to get query params
* How to use context to get function name and state
   
   
**[State Manager](./guide-statemanager)** (obsolete)
* What is statemanager and default statemanager
* How to implement and manage state with 3rd party statemanager (minioDb)
  
  
**[Hmac for security](./guide-hamc)** (obsolete)
* How HMAC can be used to secure communication
   
    
**[Tracing with Opentracing](./guide-opentracing)** (obsolete)
* How faasflow can be traced with opentracing
