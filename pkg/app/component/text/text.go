// Package text fits and decomposes the strings the interface displays.
package text

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Ellipsis marks a value which did not fit.
const Ellipsis = "…"

// Fit a line to a width in terminal cells.
//
// ANSI aware, which the hand rolled version this replaced was not: it counted
// runes, so a styled value was measured with its escape sequences and got cut
// inside one, and a wide character was measured as one cell and overflowed.
func Fit(s string, width int) string {
	if width <= 0 {
		return ""
	}

	if ansi.StringWidth(s) <= width {
		return s
	}

	return ansi.Truncate(s, width, Ellipsis)
}

// FitMiddle a line to a width, taking the cut out of the middle.
//
// For a value whose ends are what identify it and whose middle is filler — a
// long path, where the tail is the file you care about and the head is the
// mount point — cutting the tail off throws away the informative half.
func FitMiddle(s string, width int) string {
	if width <= 0 {
		return ""
	}

	if ansi.StringWidth(s) <= width {
		return s
	}

	if width <= len(Ellipsis) {
		return Ellipsis
	}

	// The tail gets the odd cell: it is the end which usually names the thing.
	var (
		head = (width - 1) / 2
		tail = width - 1 - head
	)

	return ansi.Truncate(s, head, "") + Ellipsis + ansi.TruncateLeft(s, ansi.StringWidth(s)-tail, "")
}

// Identifier splits a fully qualified name into the parts worth weighting
// differently.
//
// A Drupal method name is mostly namespace, and the namespace is mostly the
// same on every row. Splitting it lets the namespace recede and the member
// carry the emphasis, so the eye lands on the part it was hunting for.
//
//	Drupal\Core\Render\Renderer::renderRoot
//	└─ namespace ────┘└─ class ┘└─ member ─┘
//
// A name with no namespace or no member returns those parts empty rather than
// guessing, so a plain function like curl_exec is left alone.
func Identifier(name string) (namespace, class, member string) {
	rest := name

	if at := strings.LastIndex(rest, "::"); at >= 0 {
		member, rest = rest[at:], rest[:at]
	}

	if at := strings.LastIndex(rest, `\`); at >= 0 {
		namespace, class = rest[:at+1], rest[at+1:]
	} else {
		class = rest
	}

	return namespace, class, member
}

// Abbreviate a namespace to the initial of each of its parts.
//
//	Drupal\Core\Render\  ->  D\C\R\
//
// Which keeps enough to tell Drupal core from a contrib module at a glance,
// for a third of the width.
func Abbreviate(namespace string) string {
	if namespace == "" {
		return ""
	}

	parts := strings.Split(strings.TrimSuffix(namespace, `\`), `\`)

	var b strings.Builder

	for _, part := range parts {
		if part == "" {
			continue
		}

		b.WriteString(string([]rune(part)[0]))
		b.WriteString(`\`)
	}

	return b.String()
}
