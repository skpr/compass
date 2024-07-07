package envget

import "os"

// GetBool from environment variable with fallback.
func GetBool(key string, fallback bool) bool {
	val := os.Getenv(key)

	if val == "true" {
		return true
	}

	if val == "false" {
		return false
	}

	return fallback
}
