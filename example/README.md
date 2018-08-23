
### Overview
We want to create a chain that handles profile pic upload login for a dating app that mandate the user to use picture with only once face and color.

To achive that we use three functions

| Function |  description |
| ---- | ----- |
|facedetect | detect the no of face in a picture |
| colorization | colorize black and white picture |
| image-resizer | resize image to 20% of its size |

### Writing Function
We use two different kind of function  
**Sync** and **aSync**

#### Writing Sync Function `upload-chain`
Sync function meant to perform all the operation in Sync and reply the caller once finished

##### Define Chain:
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
				return context.GetPhaseInput(), nil  // return the initial data
			default:
				return nil, fmt.Errorf("More than one face detected, picture should have single face")
			}
			return nil, nil
		}).
		ApplyAsync("colorization", map[string]string{"method": "post"}, nil).
		ApplyAsync("image-resizer", map[string]string{"method": "post"}, nil)
```

#### Writing ASync Function `upload-chain-async`
ASync function meant to perform all the operation in aSync and upload the result in a storage

##### Define Chain:
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
				return context.GetPhaseInput(), nil // return the initial data
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
			err = Upload(client, "http://gateway:8080/function/file-storage", "chris.jpg", r) // upload to storage
			if err != nil {
				return nil, err
			}
			return nil, nil
		})
```

#### Invoke sync function `upload-chain`
    
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
