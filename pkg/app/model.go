// Package app for handling the main application.
package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"

	"github.com/skpr/compass/pkg/app/events"
)

// NewModel for executing this application.
func NewModel(probePath, ollamaURL, ollamaModel string) *Model {
	return &Model{
		ProbePath:    probePath,
		OllamaURL:    ollamaURL,
		OllamaModel:  ollamaModel,
		summaryCache: make(map[string]string),
	}
}

// Model for storing the state of the application.
type Model struct {
	// Path to the compass.so we are probing.
	ProbePath string

	// Ollama configuration.
	OllamaURL   string
	OllamaModel string

	// The current display that is selected.
	PageSelected Page

	// Dimensions.
	Height int
	Width  int

	// Storage.
	Current *events.Trace
	Traces  map[string]events.Trace

	// AI Summary overlay state.
	showSummary    bool
	summaryText    string
	summaryLoading bool
	summaryError   string
	summaryCache   map[string]string

	// Models.
	search   list.Model
	metadata table.Model
	spans    table.Model
	totals   table.Model
	logs     list.Model
}
