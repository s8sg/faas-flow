package faasflowMinioStateManager

import (
	"bytes"
	"fmt"
	minio "github.com/minio/minio-go"
	faasflow "github.com/s8sg/faasflow"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

type MinioStateManager struct {
	bucketName  string
	flowName    string
	requestId   string
	minioClient *minio.Client
}

// GetMinioStateManager Initialize a minio StateManager object based on configuration
// Depends on s3_url, s3-secret-key, s3-access-key, [s3_bucket, s3_region](optional), workflow_name
func GetMinioStateManager() (faasflow.StateManager, error) {

	minioStateManager := &MinioStateManager{}

	region := regionName()
	bucketName := bucketName()

	minioClient, connectErr := connectToMinio(region)
	if connectErr != nil {
		return nil, fmt.Errorf("Failed to initialize minio, error %s", connectErr.Error())
	}

	minioClient.MakeBucket(bucketName, region)
	minioStateManager.bucketName = bucketName

	minioStateManager.minioClient = minioClient

	return minioStateManager, nil
}

func (minioState *MinioStateManager) Init(flowName string, requestId string) error {
	minioState.flowName = flowName
	minioState.requestId = requestId

	return nil
}

func (minioState *MinioStateManager) Set(key string, value string) error {
	if minioState.minioClient == nil {
		return fmt.Errorf("minio client not initialized, use GetMinioStateManager()")
	}

	fullPath := getPath(minioState.bucketName, minioState.flowName, minioState.requestId, key)
	reader := bytes.NewReader([]byte(value))
	_, err := minioState.minioClient.PutObject(minioState.bucketName,
		fullPath,
		reader,
		int64(reader.Len()),
		minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("error writing: %s, error: %s", fullPath, err.Error())
	}

	return nil
}

func (minioState *MinioStateManager) Get(key string) (string, error) {
	if minioState.minioClient == nil {
		return "", fmt.Errorf("minio client not initialized, use GetMinioStateManager()")
	}

	fullPath := getPath(minioState.bucketName, minioState.flowName, minioState.requestId, key)
	obj, err := minioState.minioClient.GetObject(minioState.bucketName, fullPath, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("error reading: %s, error: %s", fullPath, err.Error())
	}

	data, _ := ioutil.ReadAll(obj)

	return string(data), nil
}

func (minioState *MinioStateManager) Del(key string) error {
	if minioState.minioClient == nil {
		return fmt.Errorf("minio client not initialized, use GetMinioStateManager()")
	}

	fullPath := getPath(minioState.bucketName, minioState.flowName, minioState.requestId, key)
	err := minioState.minioClient.RemoveObject(minioState.bucketName, fullPath)
	if err != nil {
		return fmt.Errorf("error removing: %s, error: %s", fullPath, err.Error())
	}
	return nil
}

func readSecret(key string) (string, error) {
	basePath := "/var/openfaas/secrets/"
	if len(os.Getenv("secret_mount_path")) > 0 {
		basePath = os.Getenv("secret_mount_path")
	}

	readPath := path.Join(basePath, key)
	secretBytes, readErr := ioutil.ReadFile(readPath)
	if readErr != nil {
		return "", fmt.Errorf("unable to read secret: %s, error: %s", readPath, readErr)
	}
	val := strings.TrimSpace(string(secretBytes))
	return val, nil
}

func connectToMinio(region string) (*minio.Client, error) {

	endpoint := os.Getenv("s3_url")

	tlsEnabled := tlsEnabled()

	secretKey, err := readSecret("s3-secret-key")
	accessKey, err := readSecret("s3-access-key")
	if err != nil {
		return nil, err
	}

	return minio.New(endpoint, accessKey, secretKey, tlsEnabled)
}

// getPath produces a string as <bucketname>/<flowname>/<requestId>/<key>.value
func getPath(bucket, flowName, requestId, key string) string {
	fileName := fmt.Sprintf("%s.value", key)
	return fmt.Sprintf("%s/%s/%s/%s", bucket, flowName, requestId, fileName)
}

func tlsEnabled() bool {
	if connection := os.Getenv("s3_tls"); connection == "true" || connection == "1" {
		return true
	}
	return false
}

func bucketName() string {
	bucketName, exist := os.LookupEnv("s3_bucket")
	if exist == false || len(bucketName) == 0 {
		bucketName = "pipeline"
		log.Printf("Bucket name not found, set to default: %v\n", bucketName)
	}
	return bucketName
}

func regionName() string {
	regionName, exist := os.LookupEnv("s3_region")
	if exist == false || len(regionName) == 0 {
		regionName = "us-east-1"
	}
	return regionName
}
