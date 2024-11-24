package elf

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// SystemTapNote represents a parsed SystemTap note with provider, name, and arguments
type SystemTapNote struct {
	Provider string
	Name     string
	Args     []string
}

// parseReadelfNotes parses the output of "readelf -n" and extracts SystemTap probe arguments
func ReadNotes(input string) ([]SystemTapNote, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	var notes []SystemTapNote

	// Regex patterns to match relevant lines
	providerPattern := regexp.MustCompile(`^\s*Provider:\s*(\S+)`)
	namePattern := regexp.MustCompile(`^\s*Name:\s*(\S+)`)
	argsPattern := regexp.MustCompile(`^\s*Arguments:\s*(.+)`)

	var currentNote *SystemTapNote

	for scanner.Scan() {
		line := scanner.Text()

		if match := providerPattern.FindStringSubmatch(line); match != nil {
			// If a new provider is found, finalize the previous note and start a new one
			if currentNote != nil {
				notes = append(notes, *currentNote)
			}
			currentNote = &SystemTapNote{
				Provider: match[1],
			}
		} else if match := namePattern.FindStringSubmatch(line); match != nil && currentNote != nil {
			currentNote.Name = match[1]
		} else if match := argsPattern.FindStringSubmatch(line); match != nil && currentNote != nil {
			args := strings.Split(match[1], " ")
			currentNote.Args = args
		}
	}

	// Add the last note if exists
	if currentNote != nil {
		notes = append(notes, *currentNote)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return notes, nil
}
