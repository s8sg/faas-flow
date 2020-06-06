package config

import (
	"os"
	"time"
)

func WriteTimeout() time.Duration {
	return parseIntOrDurationValue(os.Getenv("write_timeout"), 10*time.Second)
}
