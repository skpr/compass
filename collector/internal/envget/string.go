package envget

import "os"

// GetString from environment variable with fallback.
func GetString(key, fallback string) string {
	val := os.Getenv(key)

	if val != "" {
		return val
	}

	return fallback
}
