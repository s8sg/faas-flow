package ConsulStateStore

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	faasflow "github.com/s8sg/faas-flow"
)

type ConsulStateStore struct {
	consulKeyPath string
	consulClient  *consul.Client
	kv            *consul.KV

	RetryCount int
}

func GetConsulStateStore(address, datacenter string) (faasflow.StateStore, error) {

	consulST := &ConsulStateStore{}

	config := consul.DefaultConfig()
	if address != "" {
		config.Address = address
	}
	if datacenter != "" {
		config.Datacenter = datacenter
	}

	cc, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	consulST.consulClient = cc
	consulST.kv = cc.KV()

	consulST.RetryCount = 10

	return consulST, nil
}

// Configure
func (consulStore *ConsulStateStore) Configure(flowName string, requestId string) {
	consulStore.consulKeyPath = fmt.Sprintf("faasflow/%s/%s", flowName, requestId)
}

// Init (Called only once in a request)
func (consulStore *ConsulStateStore) Init() error {
	return nil
}

// Update Compare and Update a valuer
func (consulStore *ConsulStateStore) Update(key string, oldValue string, newValue string) error {
	key = consulStore.consulKeyPath + "/" + key
	pair, _, err := consulStore.kv.Get(key, nil)
	if err != nil {
		return fmt.Errorf("failed to get key %s, error %v", key, err)
	}
	if pair == nil {
		return fmt.Errorf("failed to get key %s", key)
	}
	if string(pair.Value) != oldValue {
		return fmt.Errorf("Old value doesn't match for key %s", key)
	}
	modifyIndex := pair.ModifyIndex

	p := &consul.KVPair{Key: key, Value: []byte(newValue), ModifyIndex: modifyIndex}
	_, err = consulStore.kv.Put(p, nil)
	if err != nil {
		return fmt.Errorf("failed to update key %s, error %v", key, err)
	}
	return nil
}

// Set Sets a value (override existing, or create one)
func (consulStore *ConsulStateStore) Set(key string, value string) error {
	key = consulStore.consulKeyPath + "/" + key
	p := &consul.KVPair{Key: key, Value: []byte(value)}
	_, err := consulStore.kv.Put(p, nil)
	if err != nil {
		return fmt.Errorf("failed to set key %s, error %v", key, err)
	}
	return nil
}

// Get Gets a value
func (consulStore *ConsulStateStore) Get(key string) (string, error) {
	key = consulStore.consulKeyPath + "/" + key
	pair, _, err := consulStore.kv.Get(key, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get %s, error %v", key, err)
	}
	if pair == nil {
		return "", fmt.Errorf("failed to get %s, returned nil", key)
	}
	return string(pair.Value), nil
}

// Cleanup (Called only once in a request)
func (consulStore *ConsulStateStore) Cleanup() error {
	_, err := consulStore.kv.DeleteTree(consulStore.consulKeyPath, nil)
	if err != nil {
		return fmt.Errorf("error removing %s, error %v", consulStore.consulKeyPath, err)
	}
	return nil
}
