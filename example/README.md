
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
      // define Pipleline	
      flow.
                Modify(func(data []byte) ([]byte, error) {
                        context.Set("rawImage", data)
                        return data, nil
                }).
                Apply("facedetect", faasflow.Sync).
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
                                data, err := context.GetBytes("rawImage")
                                if err != nil {
                                        return nil, fmt.Errorf("Failed to retrive picture from state, error %v", err)
                                }
                                return data, nil
                        }
                        return nil, fmt.Errorf("More than one face detected, picture should contain single face")
                }).
                Apply("colorization", faasflow.Sync).
                Apply("image-resizer", faasflow.Sync).
                OnFailure(func(err error) ([]byte, error) {
                        log.Printf("Failed to upload picture for request id %s, error %v",
                                context.GetRequestId(), err)
                        errdata := fmt.Sprintf("{\"error\": \"%s\"}", err.Error())

                        return []byte(errdata), err
                })

```

#### Writing ASync Function `upload-flow-async`
ASync function meant to perform all the operation in aSync and upload the result in a storage

##### Define Chain:
Function definition
```go
      // initialize minio DataStore
      miniosm, err := minioDataStore.GetMinioDataStore()
      if err != nil {
            return err
      }
      // Set DataStore
      context.SetDataStore(miniosm)

      // define Pipleline
      flow.
                Modify(func(data []byte) ([]byte, error) {
                        // Set the name of the file (error if not specified)
                        filename := getQuery("file")
                        if filename != "" {
                                context.Set("fileName", filename)
                        } else {
                                return nil, fmt.Errorf("Provide file name with `--query file=<name>`")
                        }
                        // Set data to reuse after facedetect
                        err := context.Set("rawImage", data)
                        if err != nil {
                                return nil, fmt.Errorf("Failed to upload picture to state, error %v", err)
                        }
                        return data, nil
                }).
                Apply("facedetect").
                Modify(func(data []byte) ([]byte, error) {
                        // validate face
                        err := validateFace(data)
                        if err != nil {
                                file, _ := context.GetString("fileName")
                                return nil, fmt.Errorf("File %s, %v", file, err)
                        }
                        // Get data from context
                        rawdata, err := context.GetBytes("rawImage")
                        if err != nil {
                                return nil, fmt.Errorf("Failed to retrive picture from state, error %v", err)
                        }
                        return rawdata, err
                }).
                Apply("colorization").
                Apply("image-resizer").
                Modify(func(data []byte) ([]byte, error) {
                        // get file name from context
                        filename, err := context.GetString("fileName")
                        if err != nil {
                                return nil, fmt.Errorf("Failed to get file name in context, %v", err)
                        }
                        // upload file to storage
                        err = upload(&http.Client{}, "http://gateway:8080/function/file-storage",
                                filename, bytes.NewReader(data))
                        if err != nil {
                                return nil, err
                        }
                        return nil, nil
                }).
                OnFailure(func(err error) ([]byte, error) {
                        log.Printf("Failed to upload picture for request id %s, error %v",
                                context.GetRequestId(), err)
                        errdata := fmt.Sprintf("{\"error\": \"%s\"}", err.Error())

                        return []byte(errdata), err
                }).
                Finally(func(state string) {
                        // Optional (cleanup)
                        // Cleanup is not needed if using default DataStore
                        context.Del("fileName")
                        context.Del("rawImage")
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
