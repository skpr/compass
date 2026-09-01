package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/functioncalls"
)

func TestLoadConfig_Defaults(t *testing.T) {
	config, err := loadConfig("")
	require.NoError(t, err)

	assert.Equal(t, ":28624", config.Addr)
	assert.Equal(t, "php-fpm", config.PHPProcessName)
	assert.Equal(t, "/usr/lib/php/modules/compass.so", config.PHPExtensionPath)
	assert.Equal(t, "node", config.NodeProcessName)
	assert.Equal(t, "/usr/lib/compass/node/compass.node", config.NodeAddonPath)
	assert.Equal(t, time.Minute, config.DiscoveryTimeout)
	assert.Equal(t, functioncalls.DefaultMax, config.MaxFunctionCalls)
}

func TestLoadConfig_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sidecar.yaml")

	require.NoError(t, os.WriteFile(path, []byte(`addr: ":9000"
log_level: "debug"
php_process_name: "php-fpm8"
discovery_timeout: "15s"
max_function_calls: 2500
token: "xxxyyyzzz"
`), 0600))

	config, err := loadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, ":9000", config.Addr)
	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, "php-fpm8", config.PHPProcessName)
	assert.Equal(t, 15*time.Second, config.DiscoveryTimeout)
	assert.Equal(t, 2500, config.MaxFunctionCalls)
	assert.Equal(t, "xxxyyyzzz", config.Token)

	// Values which are absent from the file still get their default.
	assert.Equal(t, "/usr/lib/compass/node/compass.node", config.NodeAddonPath)
}

func TestLoadConfig_EnvironmentOverridesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sidecar.yaml")

	require.NoError(t, os.WriteFile(path, []byte("addr: \":9000\"\n"), 0600))

	t.Setenv("COMPASS_SIDECAR_ADDR", ":7000")
	t.Setenv("COMPASS_SIDECAR_MAX_FUNCTION_CALLS", "750")

	config, err := loadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, ":7000", config.Addr)
	assert.Equal(t, 750, config.MaxFunctionCalls)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	assert.Error(t, err)
}
