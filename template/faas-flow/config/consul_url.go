package config

import (
	"os"
)

func ConsulURL() string {
	val := os.Getenv("consul_url")
	if len(val) == 0 {
		val = "consul.faasflow:8500"
	}
	return val
}
