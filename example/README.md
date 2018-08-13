

#### Getting started
Build and deploy the stack
```
make
```
#### Invoke Sync function `upload-chain`  
Function definition
```go
        chain.Apply("colorization", map[string]string{"method": "post"}, nil).
                Apply("image-resizer", map[string]string{"method": "post"}, nil)
```

Invoke chain
```bash
cat apollo13.jpg | faas-cli invoke -f stack.yml upload-chain > apollo13-compressed.jpg
``` 
   
   
#### Invoke Async function `upload-chain-async`  
Function definition
```go
        chain.Apply("colorization", map[string]string{"method": "post"}, nil).
                ApplyAsync("image-resizer", map[string]string{"method": "post"}, nil).
                ApplyModifier(func(data []byte) ([]byte, error) {
                        client := &http.Client{}
                        r := bytes.NewReader(data)
                        err = Upload(client, "http://gateway:8080/function/file-storage", "apollo13.jpg", r)
                        if err != nil {
                                return nil, err
                        }
                        return nil, nil
                })
```
Invoke chain
```bash
cat apollo13.jpg | faas-cli invoke --async -f stack.yml upload-chain-async
``` 
Download from storage    
```bash
curl http://127.0.0.1:8080/function/file-storage?file=apollo13.jpg > apollo13-compressed-async.jpg
```
