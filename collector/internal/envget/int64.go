package envget

import (
	"os"
	"strconv"
)

// Int64 returns an int env var.
func Int64(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		v, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}

		return int64(v)
	}

	return fallback
}
