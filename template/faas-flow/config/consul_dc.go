package config

import (
	"os"
)

func ConsulDC() string {
	val := os.Getenv("consul_dc")
	if len(val) == 0 {
		val = "dc1"
	}
	return val
}
