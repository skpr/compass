package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatches_UsesCaseInsensitiveSubstringMatching(t *testing.T) {
	values := []string{
		"GET /en/recipes",
		"GET /sites/default/files/styles/large_3_2_2x/public/pizza-umami.jpg.webp?itok=89asmEfV",
		"GET /sites/default/files/styles/large_3_2_2x/public/mediterranean-quiche-umami.jpg.webp?itok=vCwBS68x",
		"GET /EN/RECIPES/VEGAN",
		"GET /health",
	}

	assert.Equal(t, []int{0, 3}, matches(values, "recipes"))
	assert.Equal(t, []int{0, 3}, matches(values, "RECIPES"))
}

func TestMatches_EmptyQueryMatchesEverything(t *testing.T) {
	assert.Equal(t, []int{0, 1}, matches([]string{"first", "second"}, " \t"))
}
