// Package envget is used to get environment variables with fallbacks.
package envget

import "os"

// String from environment variable with fallback.
func String(key, fallback string) string {
	val := os.Getenv(key)

	if val != "" {
		return val
	}

	return fallback
}
