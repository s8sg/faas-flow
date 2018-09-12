
### Overview
We want to create a flow that handles profile pic upload login for a dating app that mandate the user to use picture with only once face and color.

To achive that we use three functions

| Function |  description |
| ---- | ----- |
|facedetect | detect the no of face in a picture |
| colorization | colorize black and white picture |
| image-resizer | resize image to 20% of its size |

### Writing Function
We use two different kind of function  
**Sync** and **aSync**

#### Writing Sync Function `upload-pipeline`
Sync function meant to perform all the operation in Sync and reply the caller once finished

##### Define Chain:
```go
	flow.Apply("facedetect", faasflow.Sync).
		Modify(func(data []byte) ([]byte, error) {
			result := FaceResult{}
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode facedetect result, error %v", err)
			}
			switch len(result.Faces) {
			case 0:
				return nil, fmt.Errorf("No face detected, picture should contain one face")
			case 1:
				return faasflow.GetContext().GetPhaseInput(), nil
			}
			return nil, fmt.Errorf("More than one face detected, picture should have single face")
		}).
		Apply("colorization", faasflow.Sync).
		Apply("image-resizer", faasflow.Sync)
```

#### Writing ASync Function `upload-flow-async`
ASync function meant to perform all the operation in aSync and upload the result in a storage

##### Define Chain:
Function definition
```go
	flow.Apply("facedetect").
		Modify(func(data []byte) ([]byte, error) {
			result := FaceResult{}
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode facedetect result, error %v", err)
			}
			switch len(result.Faces) {
			case 0:
				return nil, fmt.Errorf("No face detected, picture should contain one face")
			case 1:
				return faasflow.GetContext().GetPhaseInput(), nil
			}
			return nil, fmt.Errorf("More than one face detected, picture should have single face")
		}).
		Apply("colorization").
		Apply("image-resizer").
		Modify(func(data []byte) ([]byte, error) {
			client := &http.Client{}
			r := bytes.NewReader(data)
			err = Upload(client, "http://gateway:8080/function/file-storage", "chris.jpg", r)
			if err != nil {
				return nil, err
			}
			return nil, nil
		})
```

#### Invoke sync function `upload-flow`
    
##### Invoke flow with a image with more than once face
```bash
cat coldplay.jpg | faas-cli invoke -f stack.yml upload-pipeline
``` 
It will result in an error: `More than one face detected, picture should have single face`

##### Invoke flow with right image
```bash
cat chris.jpg | faas-cli invoke -f stack.yml upload-pipeline > chris-dp.jpg
``` 
It will create a color and compressed image
     
     
#### Invoke Async function `upload-pipeline-async`  

##### Invoke flow with a image with more than once face
```bash
cat coldplay.jpg | faas-cli invoke --query file=coldplay.jpg --async -f stack.yml upload-pipeline-async
``` 
It will result in an error: `More than one face detected, picture should have single face`
      
##### Invoke flow with right image
```bash
cat chris.jpg | faas-cli invoke --query file=chris.jpg --async -f stack.yml upload-pipeline-async
```  
Download from the storage    
```bash
curl http://127.0.0.1:8080/function/file-storage?file=chris.jpg > chris-dp.jpg
```
