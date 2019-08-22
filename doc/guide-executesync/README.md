### [Execute Functions](./example)

#### Create two dummy functions `func1` and `func2`
We will create two very simple dummy functions with python template and execute them from faasflow. 
```bash
faas new --lang python func1
mv func1.yml stack.yml
faas new --lang python -a stack.yml func2
```
Edit the function `func1` at `func1/handler.py`
```python
def handle(req):
    return "func1(" + req + ")"
```

Edit the function `func2`  at `func2/handler.py`
```python
def handler(req):
    return "func2(" + req + ")"
```

Build and deploy the dummy functions
```bash
faas build -f stack.yml
faas deploy -f stack.yml
```

#### Create Faasflow function to stitch both of them
We will create a faasflow function `syncflow`, to stitch both of them in a simgle flow
```bash
faas template pull https://github.com/s8sg/faas-flow
faas new --lang faasflow -a stack.yml syncflow
```
add the faasflow mandetory variables for `syncflow` at `stack.yml`
```yaml
    environment:
       workflow_name: "helloworld"
       gateway: "gateway:8080"
       enable_tracing: false
       enable_hmac: false
       write_debug: true
       combine_output: false
```

`syncflow` will call both `func1` and `func2` in sync and return the response to the caller.  

#### Call function using `Apply()`
`Apply()` calls allow to call a faas function. Although by default `Apply()` is `async`. In order to call function in sync one need to pass the `faasflow.Sync` option.   
Implement the `Define()` in `syncflow/handler.go`:
```go
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
        flow.Apply("func1", faasflow.Sync).Apply("func2", faasflow.Sync)
        return
}
```
   
   
Build and deploy the faasflow function with dummy functions
```bash
faas build -f stack.yml
faas deploy -f stack.yml
```
    
     
Call the faasflow
```bash
echo bingo | faas invoke -f stack.yml syncflow
```

#### Track request with the Request Id

Each request to a faasflow is assigned with a auto generated unique request Id. A request can be traced or debuged using the request Id from faasflow log. To get the logs from `syncflow` function 
```bash
docker service logs syncflow 

syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:57:33 Version: 0.8.0	SHA: 829262e493baf739fbd1c75d0ee5e853d15c7561
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:57:33 Writing lock-file to: /tmp/.lock
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 Forking fprocess.
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 Query
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 Path  /function/syncflow
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | func2(func1(bingo))
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 stderr: 2018/10/23 07:58:30 tracing is disabled
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 Failed to load faasflow-hmac-secret using default
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 [Request `bf7d99n858qjjid666v0`] Created
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 [Request `bf7d99n858qjjid666v0`] Executing phase 0
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 [Request `bf7d99n858qjjid666v0`] Executing function `func1`
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 [Request `bf7d99n858qjjid666v0`] Executing function `func2`
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 [Request `bf7d99n858qjjid666v0`] Phase 0 completed successfully
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 [Request `bf7d99n858qjjid666v0`] Completed successfully
syncflow.1.ymd8h7958s8j@linuxkit-025000000001    | 2018/10/23 07:58:30 Duration: 0.177145 seconds
```
Here `bf7d99n858qjjid666v0` is the assigned request Id.
