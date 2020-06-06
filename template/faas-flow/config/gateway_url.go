package config

import (
	"os"
)

// GatewayURL return the gateway address from env
func GatewayURL() string {
	gateway := os.Getenv("gateway")
	if gateway == "" {
		gateway = "gateway.openfaas:8080"
	}
	return gateway
}
