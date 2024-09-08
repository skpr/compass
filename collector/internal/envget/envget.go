// Package envget is used to get environment variables with fallbacks.
package envget

import (
	"os"
	"strconv"
)

// String from environment variable with fallback.
func String(key, fallback string) string {
	val := os.Getenv(key)

	if val != "" {
		return val
	}

	return fallback
}

// Float64 returns an float64 env var.
func Float64(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(err)
		}

		return v
	}

	return fallback
}
