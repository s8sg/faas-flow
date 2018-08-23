
#### Getting started
Build and deploy the stack
```
./build; ./deploy
```
#### Invoke Sync function `upload-chain`  
Function definition
```go
	chain.Apply("facedetect", map[string]string{"method": "post"}, nil).
		ApplyModifier(func(data []byte) ([]byte, error) {
			context := faaschain.GetContext()
			result := FaceResult{}
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode facedetect result, error %v", err)
			}
			switch len(result.Faces) {
			case 0:
				return nil, fmt.Errorf("No face detected, picture should contain one face")
			case 1:
				return context.GetPhaseInput(), nil
			default:
				return nil, fmt.Errorf("More than one face detected, picture should have single face")
			}
			return nil, nil
		}).
		ApplyAsync("colorization", map[string]string{"method": "post"}, nil).
		ApplyAsync("image-resizer", map[string]string{"method": "post"}, nil)
```
    
##### Invoke chain with a image with more than once face
```bash
cat coldplay.jpg | faas-cli invoke -f stack.yml upload-chain
``` 
It will result in an error: `More than one face detected, picture should have single face`

##### Invoke chain with right image
```bash
cat chris.jpg | faas-cli invoke -f stack.yml upload-chain > chris-dp.jpg
``` 
It will create a color and compressed image
     
     
#### Invoke Async function `upload-chain-async`  
Function definition
```go
	chain.Apply("facedetect", map[string]string{"method": "post"}, nil).
		ApplyModifier(func(data []byte) ([]byte, error) {
			context := faaschain.GetContext()
			result := FaceResult{}
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode facedetect result, error %v", err)
			}
			switch len(result.Faces) {
			case 0:
				return nil, fmt.Errorf("No face detected, picture should contain one face")
			case 1:
				return context.GetPhaseInput(), nil
			default:
				return nil, fmt.Errorf("More than one face detected, picture should have single face")
			}
			return nil, nil
		}).
		ApplyAsync("colorization", map[string]string{"method": "post"}, nil).
		ApplyAsync("image-resizer", map[string]string{"method": "post"}, nil).
		ApplyModifier(func(data []byte) ([]byte, error) {
			client := &http.Client{}
			r := bytes.NewReader(data)
			err = Upload(client, "http://gateway:8080/function/file-storage", "chris.jpg", r)
			if err != nil {
				return nil, err
			}
			return nil, nil
		})
```
    
##### Invoke chain with a image with more than once face
```bash
cat coldplay.jpg | faas-cli invoke --async -f stack.yml upload-chain-async
``` 
It will result in an error: `More than one face detected, picture should have single face`
      
##### Invoke chain with right image
```bash
cat chris.jpg | faas-cli invoke --async -f stack.yml upload-chain-async
```  
Download from the storage    
```bash
curl http://127.0.0.1:8080/function/file-storage?file=chris.jpg > chris-dp.jpg
```
