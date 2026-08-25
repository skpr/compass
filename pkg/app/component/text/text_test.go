package text

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestFit(t *testing.T) {
	assert.Equal(t, "", Fit("abc", 0))
	assert.Equal(t, "", Fit("abc", -1))
	assert.Equal(t, "abc", Fit("abc", 5))
	assert.Equal(t, "abc", Fit("abc", 3))
	assert.Equal(t, "ab…", Fit("abcd", 3))
}

// The reason this replaced a rune counting version: a wide character is two
// cells, and measuring it as one overflows the row it is on.
func TestFit_WideCharacters(t *testing.T) {
	wide := strings.Repeat("世", 10)

	assert.LessOrEqual(t, ansi.StringWidth(Fit(wide, 9)), 9)
	assert.LessOrEqual(t, ansi.StringWidth(Fit(wide, 4)), 4)
}

func TestFit_StyledValueIsNeverCutInsideAnEscapeSequence(t *testing.T) {
	styled := "\x1b[38;2;255;0;0mpermanent\x1b[0m"

	fitted := Fit(styled, 5)

	assert.LessOrEqual(t, ansi.StringWidth(fitted), 5)
	// The colour it opened with has to be closed again.
	assert.Contains(t, fitted, "\x1b[")
	assert.Equal(t, "perm…", ansi.Strip(fitted))
}

func TestIdentifier(t *testing.T) {
	tests := []struct {
		name                     string
		in                       string
		namespace, class, member string
	}{
		{
			name:      "namespaced method",
			in:        `Drupal\Core\Render\Renderer::renderRoot`,
			namespace: `Drupal\Core\Render\`,
			class:     "Renderer",
			member:    "::renderRoot",
		},
		{
			name:   "plain function",
			in:     "curl_exec",
			class:  "curl_exec",
			member: "",
		},
		{
			name:   "class without a namespace",
			in:     "Foo::bar",
			class:  "Foo",
			member: "::bar",
		},
		{
			name:      "namespace without a member",
			in:        `Drupal\node\Entity\Node`,
			namespace: `Drupal\node\Entity\`,
			class:     "Node",
		},
		{name: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, class, member := Identifier(tt.in)

			assert.Equal(t, tt.namespace, namespace)
			assert.Equal(t, tt.class, class)
			assert.Equal(t, tt.member, member)

			// Whatever the split, it has to reassemble into what it came from.
			assert.Equal(t, tt.in, namespace+class+member)
		})
	}
}

func TestAbbreviate(t *testing.T) {
	assert.Equal(t, `D\C\R\`, Abbreviate(`Drupal\Core\Render\`))
	assert.Equal(t, `D\h\P\S\`, Abbreviate(`Drupal\help\Plugin\Search\`))
	assert.Equal(t, "", Abbreviate(""))
	assert.Equal(t, `D\`, Abbreviate(`Drupal\`))
}

// The whole point is that it is shorter.
func TestAbbreviate_IsShorter(t *testing.T) {
	full := `Drupal\Component\EventDispatcher\`

	assert.Less(t, len(Abbreviate(full)), len(full)/3)
}

func TestFitMiddle(t *testing.T) {
	assert.Equal(t, "", FitMiddle("abc", 0))
	assert.Equal(t, "abcdef", FitMiddle("abcdef", 10))
	assert.Equal(t, "…", FitMiddle("abcdef", 1))
}

// The point of cutting the middle is that both ends survive: for a path, the
// mount point and the file name are the halves worth keeping.
func TestFitMiddle_KeepsBothEnds(t *testing.T) {
	path := "/proc/61642/root/usr/lib/php/modules/compass.so"

	fitted := FitMiddle(path, 30)

	assert.Equal(t, 30, ansi.StringWidth(fitted))
	assert.True(t, strings.HasPrefix(fitted, "/proc"), "lost the head: %q", fitted)
	assert.True(t, strings.HasSuffix(fitted, "compass.so"), "lost the tail: %q", fitted)
	assert.Contains(t, fitted, Ellipsis)
}

func TestFitMiddle_IsAlwaysTheRequestedWidth(t *testing.T) {
	path := "/proc/61642/root/usr/lib/php/modules/compass.so"

	for width := 2; width <= 60; width++ {
		assert.LessOrEqual(t, ansi.StringWidth(FitMiddle(path, width)), width, "width=%d", width)
	}
}
