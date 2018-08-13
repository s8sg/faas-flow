# FaaSChain - FaaS pipeline as a function
* Pure FaaS
* Build Over Current Go Template
* Available as a `faaschain` template
* Use FaaS platform to Communicate


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
