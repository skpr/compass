// Package main is for our test application.
package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/cli/app"
	"github.com/skpr/compass/trace"
)

func main() {
	p := tea.NewProgram(app.NewModel(), tea.WithAltScreen())

	eg := errgroup.Group{}

	eg.Go(func() error {
		for {
			p.Send(trace.Trace{
				Metadata: trace.Metadata{
					RequestID:     uuid.New().String(),
					URI:           "/foo",
					Method:        "GET",
					StartTime:     3000000,
					EndTime:       15000000,
					ExecutionTime: 12000,
				},
				FunctionCalls: []trace.FunctionCall{
					{
						Name:      "Foo::bar",
						StartTime: 3000000,
						EndTime:   15000000,
					},
					{
						Name:      "Skpr::rocks",
						StartTime: 5000000,
						EndTime:   13000000,
					},
					{
						Name:      "Baz::boo",
						StartTime: 6000000,
						EndTime:   10000000,
					},
				},
			})

			time.Sleep(time.Second)
		}
	})

	eg.Go(func() error {
		_, err := p.Run()
		return err
	})

	err := eg.Wait()
	if err != nil {
		panic(err)
	}
}
