package app

import (
	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/component/text"
	"github.com/skpr/compass/pkg/app/theme"
)

// identifierCell renders a fully qualified name with the namespace receding and
// the member carrying the emphasis.
//
// A screen of Drupal method names is mostly namespace, and mostly the same
// namespace. Abbreviating it and weighting the three parts differently is what
// lets the eye land on the part it was looking for.
func identifierCell(name string) datatable.Cell {
	namespace, class, member := text.Identifier(name)

	if namespace == "" && member == "" {
		return datatable.Styled(class, theme.S.Cell)
	}

	return datatable.Join(
		datatable.Seg(text.Abbreviate(namespace), theme.S.Faint),
		datatable.Seg(class, theme.S.Cell),
		datatable.Seg(member, theme.S.Bold),
	)
}
