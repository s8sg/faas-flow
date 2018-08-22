
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
cat chris.jpg | faas-cli invoke -f stack.yml upload-chain > chris-dp.jpg
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
    
Invoke chain with wrong image
```bash
cat coldplay.jpg | faas-cli invoke --async -f stack.yml upload-chain-async
``` 
    
Invoke chain with right image
```bash
cat chris.jpg | faas-cli invoke --async -f stack.yml upload-chain-async
``` 
Download from storage    
```bash
curl http://127.0.0.1:8080/function/file-storage?file=chris.jpg > chris-dp.jpg
```
