package controller

import (
	"log"

	"handler/function"

	minioDataStore "github.com/faasflow/faas-flow-minio-datastore"
	"github.com/faasflow/sdk"
)

func initDataStore() (dataStore sdk.DataStore, err error) {
	dataStore, err = function.OverrideDataStore()
	if err != nil {
		return nil, err
	}
	if dataStore == nil {

		/*
			minioUrl := os.Getenv("s3_url")
			if len(minioUrl) == 0 {
				minioUrl = "minio.faasflow:9000"
			}

			minioRegion := os.Getenv("s3_region")
			if len(minioRegion) == 0 {
				minioUrl = "us-east-1"
			}

			secretKeyName := os.Getenv("s3_secret_key_name")
			if len(secretKeyName) == 0 {
				secretKeyName = "s3-secret-key"
			}

			accessKeyName := os.Getenv("s3_access_key_name")
			if len(accessKeyName) == 0 {
				accessKeyName = "s3-access-key"
			}

			tlsEnabled := false
			if connection := os.Getenv("s3_tls"); connection == "true" || connection == "1" {
				tlsEnabled = true
			}

			dataStore, err = minioDataStore.Init(minioUrl, minioRegion, secretKeyName, accessKeyName, tlsEnabled)
		*/
		dataStore, err = minioDataStore.InitFromEnv()

		log.Print("Using default data store (minio)")
	}
	return dataStore, err
}
