// Package plugin is for extending the collector.
package plugin

import (
	"fmt"
	"plugin"
)

// Load a plugin from a given path.
func Load(path string) (Interface, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup symbol: %w", err)
	}

	var compassPlugin Interface

	compassPlugin, ok := symbol.(Interface)
	if !ok {
		return nil, fmt.Errorf("unexpected type from module symbol")
	}

	err = compassPlugin.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plugin: %w", err)
	}

	return compassPlugin, nil
}
