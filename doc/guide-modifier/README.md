### [using Modifiers](./example)

Modifiers provide a way to write inline code to validate, update or alter the request or response from function in a chain. A modifier can be any function that matches the following signature
```go
type Modifier func([]byte) ([]byte, error)
```
   
Modifier takes a `[]byte` as input and produce a `[]byte` as output. If error is returned it will be treated as a failure in the pipeline execution. 
   
   
To get start with modifer let us create two test fuctions `titleize` and `format`

* `titleize` takes a **json** input with `firstname` and `lastname` and titleize the names
Input/Output
```json
{
	"Firstname" : "",
	"Lastname": ""
}
```
    
* `format` takes a titleized name provided in **xml** and encodes a username as `lastname.firstname`
Input
```xml
<user>
  <Firstname>
  
  </Firstname>
  <Lastname>  
  
  </Lastname>
</user>
```
Output
```yaml
lastname.firstname
```
    
The test functions implementation can be found [here](./example)

#### Create a Faasflow to Stitch them
* We will create a faasflow to stitch them together. Say the name of the flow function is `makeuserid`.  
```bash
faas new --lang faasflow -a stack.yml makeuserid
```
* Edit the `stack.yml` to provide the necessary inputs
```yaml
    environment:
       workflow_name: "makeuserid"
       gateway: "gateway:8080"
       enable_tracing: false
       enable_hmac: false
       write_debug: true
       combine_output: false
```

#### Using modifier to verify and modify intermidiate input
Edit the flow Definition at `makeuserid/handler.go`
```golang
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
        flow.
                Apply("titleize", faasflow.Sync).
                Modify(func(data []byte) ([]byte, error) {
                        name := struct {
                                Firstname string
                                Lastname  string
                        }{}
                        err := json.Unmarshal(data, &name)
                        if err != nil {
                                return nil, err
                        }
                        user := struct {
                                XMLName   xml.Name `xml:"user"`
                                Firstname string   `xml:"Firstname"`
                                Lastname  string   `xml:"Lastname"`
                        }{}
                        user.Firstname = name.Firstname
                        user.Lastname = name.Lastname
                        resp, _ := xml.Marshal(user)
                        return resp, nil
                }).
                Apply("format", faasflow.Sync)
        return
}
```
Here modifier is being used to Both validate the return value of `titleize` function and convert the intermidiate data to the desired xml format for function `format`. 

#### Validate the input with modifier 
One can validate the input with modifier.
```golang
        flow.
                Modify(func(data []byte) ([]byte, error) {
                        name := struct {
                                Firstname string
                                Lastname  string
                        }{}
                        err := json.Unmarshal(data, &name)
                        if err != nil {
                                return nil, err
                        }
                        if name.Firstname == "" || name.Lastname == "" {
                                return nil, fmt.Errorf("Firstname and Lastname must be provided")
                        }
                        return data, nil
                }).
                Apply("titleize", faasflow.Sync).
                Modify(func(data []byte) ([]byte, error) {
                        name := struct {
                                Firstname string
                                Lastname  string
                        }{}
                        err := json.Unmarshal(data, &name)
                        if err != nil {
                                return nil, err
                        }
                        user := struct {
                                XMLName   xml.Name `xml:"user"`
                                Firstname string   `xml:"Firstname"`
                                Lastname  string   `xml:"Lastname"`
                        }{}
                        user.Firstname = name.Firstname
                        user.Lastname = name.Lastname
                        resp, _ := xml.Marshal(user)
                        return resp, nil
                }).
                Apply("format", faasflow.Sync)
```
Here a initial modifier is added to test invalid json field and also if the vanue is empty. The Modifier return error in case the validation fails and for success it returns the input.

