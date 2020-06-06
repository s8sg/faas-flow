package config

import (
	"os"
	"time"
)

func ReadTimeout() time.Duration {
	return parseIntOrDurationValue(os.Getenv("read_timeout"), 10*time.Second)
}
