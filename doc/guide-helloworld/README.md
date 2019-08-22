### [Hello World](./example)

#### Get the faasflow template
To create a new flow `faasflow` template is needed
```bash
faas template pull https://github.com/s8sg/faas-flow
```

#### Create a new faasflow
To create a new flow `helloworld`, use the `faasflow` as a `lang`
```bash
faas new --lang faasflow helloworld
```

#### Edit the flow handler
Edit the `helloworld/handler.go` to define the flow by overriding the `Define` method
```go
// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
        flow.Modify(func(data []byte) ([]byte, error) {
                return []byte(fmt.Sprintf("Hello World! You said %s!", string(data))), nil
        })
        return
}
```
`Define()` function forwards a `faasflow.Workflow` object that defines the Workflow. 

#### Modifier - Modify() 
`flow.Modify()` is the simplist way to get started with faasflow, it takes the input as a `[]byte` and returns a modified `[]byte` in response. If error is returned it will cause the execution of the workflow to fail.

#### Provide the mandetory configuration
Faasflow needs some mandetory configuration in order to operate.   
To get started edit the `helloworld.yml` and provide the below environment:
```bash
environment:
  workflow_name: "helloworld"  
  gateway: "gateway:8080"
  enable_tracing: false
  enable_hmac: false
  write_debug: true
  combine_output: false
```
The `flow_name` is required to forward the flow request to faasflow, which is used for `async` call. `gateway` provides the endpoint to reach openfaas-gateway. `enable_tracing` and `enable_hmac` will be covered in later stage, for now it can be set to `false`.
  
The faasflow prints debugs, `combine_output` make the output without the debug statement

#### Build and deploy the flow function
Build and deploy the flow function with openfaas
```bash
faas build -f helloworld.yml
faas deploy -f helloworld.yml
```

#### Execute the faasflow function
Executes the faasflow function like any other function
```bash
echo bingo | faas invoke -f helloworld.yml helloworld
```
