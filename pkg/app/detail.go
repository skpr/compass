package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/component/text"
	"github.com/skpr/compass/pkg/app/format"
	"github.com/skpr/compass/pkg/app/theme"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/trace/drupal"
)

// Shape of the detail block.
const (
	// detailLabelWidth is the column the field names sit in.
	detailLabelWidth = 11
	// detailGap separates one field from the next.
	detailGap = 3
	// detailMinValueWidth is the narrowest a value column is worth being. Below
	// it the block drops to fewer columns rather than shredding every value.
	detailMinValueWidth = 18
	// detailMaxColumns caps how wide the block spreads. Beyond three the eye
	// has to travel too far between a name and the thing it names.
	detailMaxColumns = 3
	// detailMaxRows caps the block's height, so a very narrow terminal spends
	// its rows on the trace rather than on the description of it. At any width
	// the interface is comfortable at, every field fits inside it.
	detailMaxRows = 8
)

// field of the detail block.
type field struct {
	label string
	value string
	style lipgloss.Style
	// wide gives the field a row to itself, spanning the block.
	//
	// For the values which are long and which nobody wants an abbreviation of:
	// a request id you are going to paste into a log search is no use with its
	// last twenty characters replaced by an ellipsis.
	wide bool
}

// detailFields describing the open trace.
//
// One section, every fact in it, each with the name of what it is. The
// alternative — a dense line of values separated by dots — reads quickly once
// you already know what each position means and is unreadable before that.
func (m *Model) detailFields() []field {
	if m.Current == nil {
		return nil
	}

	t := m.Current
	ms := format.Milliseconds(t.Metadata.ExecutionTime())

	fields := []field{
		{label: "Runtime", value: string(t.Metadata.Runtime), style: runtimeStyle(t.Metadata.Runtime)},
		{label: "Source", value: string(t.Metadata.Source), style: sourceStyle(t.Metadata.Source)},
	}

	switch t.Metadata.Source {
	case trace.SourceHTTP:
		fields = append(fields,
			field{label: "Method", value: t.Metadata.HTTP.Method, style: theme.S.Cell},
			field{label: "URI", value: t.Metadata.HTTP.URI, style: theme.S.Cell, wide: true},
		)
	case trace.SourceCLI:
		fields = append(fields, field{label: "Command", value: t.Metadata.CLI.Command, style: theme.S.Cell, wide: true})
	}

	fields = append(fields,
		field{
			label: "Duration",
			value: format.Duration(t.Metadata.ExecutionTime()),
			style: theme.S.Severity(theme.ForDurationMs(ms)),
		},
		field{label: "Max memory", value: format.Bytes(t.ResourceUtilisation.MaxMemory), style: theme.S.CellDim},
		field{label: "Calls", value: fmt.Sprintf("%d", len(t.FunctionCalls)), style: theme.S.CellDim},
	)

	if t.Metadata.Identified() {
		fields = append(fields, field{label: "ID", value: t.Metadata.ID, style: theme.S.CellDim, wide: true})
	}

	if !t.IngestionTime.IsZero() {
		fields = append(fields, field{
			label: "Ingested",
			value: format.RelativeTime(t.IngestionTime, time.Now()),
			style: theme.S.CellDim,
		})
	}

	return append(fields, m.cacheabilityFields()...)
}

// cacheabilityFields describing what Drupal made of the response, when there is
// anything to say.
func (m *Model) cacheabilityFields() []field {
	summary := drupal.Unmarshal(m.Current.Trace)

	if !summary.Collected {
		return nil
	}

	value, style := "permanent", theme.S.CellFaint

	switch {
	case summary.Uncacheable:
		value = "uncacheable"
		style = theme.S.Severity(theme.LevelCritical)
	case summary.EffectiveMaxAge != trace.CacheMaxAgePermanent:
		value = "max-age " + format.MaxAge(summary.EffectiveMaxAge)
		style = theme.S.Severity(theme.ForMaxAge(summary.EffectiveMaxAge))
	}

	fields := []field{
		{label: "Cacheable", value: value, style: style},
		{label: "Cache tags", value: fmt.Sprintf("%d", len(summary.Tags)), style: theme.S.CellDim},
		{label: "Contexts", value: fmt.Sprintf("%d", len(summary.Contexts)), style: theme.S.CellDim},
	}

	// Only when it happened. A counter reading zero is a field spent saying
	// nothing went wrong.
	if summary.Dropped > 0 {
		fields = append(fields, field{
			label: "Dropped",
			value: fmt.Sprintf("%d events", summary.Dropped),
			style: theme.S.Severity(theme.LevelWarn),
		})
	}

	return fields
}

// detailColumns which fit a width, and how wide each value column is.
func detailColumns(width int) (columns, valueWidth int) {
	for columns = detailMaxColumns; columns > 1; columns-- {
		valueWidth = (width - 1 - columns*(detailLabelWidth+1) - (columns-1)*detailGap) / columns
		if valueWidth >= detailMinValueWidth {
			return columns, valueWidth
		}
	}

	return 1, max(width-1-detailLabelWidth-1, 1)
}

// split fields into the ones which share a row and the ones which take one.
func split(fields []field) (narrow, wide []field) {
	for _, f := range fields {
		if f.wide {
			wide = append(wide, f)

			continue
		}

		narrow = append(narrow, f)
	}

	return narrow, wide
}

// detailRule is the line which closes the block off from the table below it.
const detailRule = 1

// detailHeight the block needs at a width, including its closing rule.
func (m *Model) detailHeight(width int) int {
	fields := m.detailFields()
	if len(fields) == 0 {
		return 0
	}

	narrow, wide := split(fields)

	columns, _ := detailColumns(width)

	return min((len(narrow)+columns-1)/columns+len(wide), detailMaxRows) + detailRule
}

// viewDetail renders the fields describing the open trace.
//
// The long fields take a row each at the top, then the short ones are laid out
// down the columns rather than across them, so a column is a list you can read
// top to bottom instead of every third item of one.
func (m *Model) viewDetail() string {
	fields := m.detailFields()
	if len(fields) == 0 {
		return ""
	}

	narrow, wide := split(fields)

	columns, valueWidth := detailColumns(m.Width)

	rows := (len(narrow) + columns - 1) / columns

	lines := make([]string, 0, rows+len(wide))

	// The long fields lead. They are what identifies the trace — which request
	// this was, and the id you would search a log with — and putting them last
	// meant they were the first thing the row cap took away on a narrow screen,
	// which is exactly backwards.
	for _, f := range wide {
		lines = append(lines, m.padLine(" "+m.renderField(f, max(m.Width-detailLabelWidth-3, 1))))
	}

	for row := range rows {
		var b strings.Builder

		b.WriteString(" ")

		for column := range columns {
			index := column*rows + row
			if index >= len(narrow) {
				break
			}

			if column > 0 {
				b.WriteString(strings.Repeat(" ", detailGap))
			}

			b.WriteString(m.renderField(narrow[index], valueWidth))
		}

		lines = append(lines, m.padLine(b.String()))
	}

	// Closed with a rule. There is no rule above it — the tab row draws one
	// directly overhead — but without one below, the last field runs straight
	// into the table's column headings and the two read as one list.
	lines = lines[:min(len(lines), detailMaxRows)]

	rule := theme.S.RuleIdle.Render(strings.Repeat(theme.RuleLight, max(m.Width, 0)))

	return strings.Join(append(lines, rule), "\n")
}

// renderField as a name and the thing it names.
func (m *Model) renderField(f field, valueWidth int) string {
	label := theme.S.Header.Render(pad(strings.ToUpper(f.label), detailLabelWidth))

	value := text.Fit(f.value, valueWidth)

	return label + " " + f.style.Render(value) + strings.Repeat(" ", max(valueWidth-ansi.StringWidth(value), 0))
}

// runtimeStyle for a runtime.
func runtimeStyle(runtime trace.Runtime) lipgloss.Style {
	switch runtime {
	case trace.RuntimePHP:
		return theme.S.RuntimePHP
	case trace.RuntimeNode:
		return theme.S.RuntimeNode
	default:
		return theme.S.CellFaint
	}
}

// sourceStyle for a source.
func sourceStyle(source trace.Source) lipgloss.Style {
	switch source {
	case trace.SourceHTTP:
		return theme.S.SourceHTTP
	case trace.SourceCLI:
		return theme.S.SourceCLI
	default:
		return theme.S.CellFaint
	}
}
