package main

import (
	"fmt"
	"github.com/boltdb/bolt"
	"sync"
	"time"
)

type BoltStateStore struct {
	flowName  string
	requestId string
	db        *bolt.DB
	mux       sync.Mutex
}

func getDbName(flowName string) string {
	return flowName + ".db"
}

func GetBoltStateStore(flowName string) (*BoltStateStore, error) {
	ss := &BoltStateStore{}
	db, err := bolt.Open(getDbName(flowName), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open file based db, error %v", err)
	}
	ss.flowName = flowName
	ss.db = db
	return ss, nil
}

// Configure the StateStore with flow name and request ID
func (ss *BoltStateStore) Configure(flowName string, requestId string) {
	ss.requestId = requestId
}

// Initialize the StateStore (called only once in a request span)
func (ss *BoltStateStore) Init() error {

	tx, err := ss.db.Begin(true)
	if err != nil {
		return fmt.Errorf("failed to create request bucket, error %v", err)
	}
	defer tx.Rollback()

	// Use the transaction...
	_, err = tx.CreateBucket([]byte(ss.requestId))
	if err != nil {
		return fmt.Errorf("failed to create request bucket, error %v", err)
	}

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to create request bucket, error %v", err)
	}

	return nil
}

// Set a value (override existing, or create one)
func (ss *BoltStateStore) Set(key string, value string) error {
	err := ss.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ss.requestId))
		err := b.Put([]byte(key), []byte(value))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to set key, error %v", err)
	}
	return nil
}

// Get a value
func (ss *BoltStateStore) Get(key string) (string, error) {

	var value string

	err := ss.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ss.requestId))
		data := b.Get([]byte(key))
		value = string(data)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to get key, error %v", err)
	}
	return value, nil
}

// Compare and Update a value
func (ss *BoltStateStore) Update(key string, oldValue string, newValue string) error {

	err := ss.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ss.requestId))
		value := b.Get([]byte(key))
		if string(value) != oldValue {
			return fmt.Errorf("value doesn't match")
		}
		err := b.Put([]byte(key), []byte(newValue))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update key, error %v", err)
	}
	return nil
}

// Cleanup all the resources in StateStore (called only once in a request span)
func (ss *BoltStateStore) Cleanup() error {

	tx, err := ss.db.Begin(true)
	if err != nil {
		return fmt.Errorf("failed to delete request bucket, error %v", err)
	}
	defer tx.Rollback()

	// Use the transaction...
	err = tx.DeleteBucket([]byte(ss.requestId))
	if err != nil {
		return fmt.Errorf("failed to delete request bucket, error %v", err)
	}

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to delete request bucket, error %v", err)
	}

	return nil
}
