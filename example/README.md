

#### Invoke Sync function `upload-chain`  
```bash
cat apollo13.jpg | faas-cli invoke -f stack.yml upload-chain > apollo13-compressed.jpg
```
   
#### Invoke Async function `upload-chain-async`  
```bash
cat apollo13.jpg | faas-cli invoke --async -H "X-Callback-Url=http://gateway:8080/function/file-storage" -f stack.yml upload-chain-async
```
    
Download from storage    
```bash
curl http://127.0.0.1:8080/function/file-storage?file=apollo13.jpg > apollo13-compressed.jpg
```
