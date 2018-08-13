# FaaSChain - FaaS pipeline as a function
* Pure FaaS
* Build Over Current Go Template
* Available as a `faaschain` template
* Use FaaS platform to Communicate
     
## What is it ?
FaaSChain allow you to define your pipeline and host it as a function
![alt overview](https://github.com/s8sg/faaschain/blob/master/doc/figure1.jpeg)
     
## How does it work ?
FaaSChain runs five mejor steps to define and run the pipeline
![alt internal](https://github.com/s8sg/faaschain/blob/master/doc/figure2.jpeg)

| phase |  description |
| ---- | ----- |
| Build Chain | Identify a request and build a chain. A incoming request could be a half finished pipeline or a fresh request. In case its not a fresh request, faas-chain parse and understand the state of the pipeline from the incoming request |
| Get Definition | FaaSChain is stateless, to get the chain defintion it calls the exposed `handler.go` every time to get the user defintion of the chain |
| Plan | FaasChain create simple plan with multiple phases. Each Phase have one or Multiple Function Request or Modifier. Once a phase is complete FaasChain asyncronously forward the request to same chain via gateway |
| Execute | Execute executes a phase by calling Modifier, FaaS-Functions or Callback. During Execution FaasChain can split a phase into two or more if it take more time |
| Repeat Or Response | In the reapeat or response phase If pipeline is not yet completed, FaasChain forwards the remaining Pipeline to the same chain via gateway. If its completed faas-chain returns the response to gateway if a `sync` request | 
  
## Example
https://github.com/s8sg/faaschain/tree/master/example
     
## Getting Started

#### **Get the `faaschain` template with `faas-cli`**.  
```
faas-cli template pull https://github.com/s8sg/faaschain
```
   
#### **Create a new `func` with `faaschain` template**.  
```bash
faas-cli new test-chain --lang faaschain
```
   
#### **Edit the `test-chain.yml` as:**.  
```yaml
  test-chain:
    lang: faaschain
    handler: ./test-chain
    image: test-chain:latest
    environment:
      gateway: "gateway:8080"
      chain_name: "test-chain"
      read_timeout: 120
      write_timeout: 120
      write_debug: true
      combine_output: false
```
> *we will discuss the meaning of the additional parameter in details  
>  in the below section
    
#### **Edit the `test-chain/handler.go`:**.  
```go
    chain.Apply("myfunc1", map[string]string{"method": "post"}, nil).
         .ApplyModifier(func(data []byte) ([]byte, error) {
                log.Printf("Making data serialized for myfunc2")
                return []byte(fmt.Sprintf("{ \"data\" : \"%s\" }", string(data))), nil
        }).Apply("myfunc2", map[string]string{"method": "post"}, nil)
```
#### **Build and Deploy the `test-chain` :**.  
Build
```bash
faas-cli build -f test-chain.yml
```
Deploy
```bash
faas-cli deploy -f test-chain.yml
```

#### **Invoke :**
```
cat data | faas-cli invoke -f test-chain.yml test-chain > updated_data
```
       
         
          
## Use async function
FaaSChain supports async function. It uses the `openfaas` async feature to implement.
To implement a async call use `ApplyAsync` function
#### **Edit the `test-chain/handler.go`:**.  
```go
    chain.Apply("myfunc1", map[string]string{"method": "post"}, nil).
         .ApplyModifier(func(data []byte) ([]byte, error) {
                log.Printf("Making data serialized for myfunc2")
                return []byte(fmt.Sprintf("{ \"data\" : \"%s\" }", string(data))), nil
        }).ApplyAsync("myfunc2", map[string]string{"method": "post"}, nil).
        Callback("http://gateway:8080/function/send2slack", 
                 map[string]string{"method": "post"}, 
                 map[string]string{"authtoken": os.Getenv(token)})
```
#### **Invoke :**
```
cat data | faas-cli invoke --async -f test-chain.yml test-chain
Function submitted asynchronously.
```

## Request Tracking
Request can be tracked from the log by `RequestId`. For each new Request a unique `RequestId` is generated. 
```bash
2018/08/13 07:51:59 [request `bdojh7oi7u6bl8te4r0g`] Created
2018/08/13 07:52:03 [Request `bdojh7oi7u6bl8te4r0g`] Received
```
