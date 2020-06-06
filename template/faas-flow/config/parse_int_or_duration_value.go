package config

import (
	"strconv"
	"time"
)

func parseIntOrDurationValue(val string, fallback time.Duration) time.Duration {
	if len(val) > 0 {
		parsedVal, parseErr := strconv.Atoi(val)
		if parseErr == nil && parsedVal >= 0 {
			return time.Duration(parsedVal) * time.Second
		}
	}

	duration, durationErr := time.ParseDuration(val)
	if durationErr != nil {
		return fallback
	}
	return duration
}
