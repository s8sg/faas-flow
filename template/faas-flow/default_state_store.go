package main

import (
	"fmt"
	"sync"
)

type DefaultStateStore struct {
	flowName     string
	requestId    string
	keyDecorator func(string) string
	cache        map[string]string
	mux          sync.Mutex
}

// Configure the StateStore with flow name and request ID
func (ss *DefaultStateStore) Configure(flowName string, requestId string) {
	ss.keyDecorator = func(key string) string {
		dKey := flowName + "-" + requestId + "-" + key
		return dKey
	}
}

// Initialize the StateStore (called only once in a request span)
func (ss *DefaultStateStore) Init() error {
	ss.cache = make(map[string]string)
	return nil
}

// Set a value (override existing, or create one)
func (ss *DefaultStateStore) Set(key string, value string) error {
	ss.mux.Lock()
	ss.cache[ss.keyDecorator(key)] = value
	ss.mux.Unlock()
	return nil
}

// Get a value
func (ss *DefaultStateStore) Get(key string) (string, error) {
	ss.mux.Lock()
	value, ok := ss.cache[ss.keyDecorator(key)]
	ss.mux.Unlock()
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return value, nil
}

// Compare and Update a value
func (ss *DefaultStateStore) Update(key string, oldValue string, newValue string) error {
	ss.mux.Lock()
	value, ok := ss.cache[ss.keyDecorator(key)]
	if !ok {
		ss.mux.Unlock()
		return fmt.Errorf("key not found")
	}
	if value != oldValue {
		ss.mux.Unlock()
		return fmt.Errorf("value doesn't match")
	}
	ss.cache[ss.keyDecorator(key)] = newValue
	ss.mux.Unlock()
	return nil
}

// Cleanup all the resources in StateStore (called only once in a request span)
func (ss *DefaultStateStore) Cleanup() error {
	return nil
}
