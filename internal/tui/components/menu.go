package components

import (
	"strings"

	"github.com/Eliasf1912/go-project/internal/tui/styles"
)

type Menu struct {
	items  []string
	cursor int
}

func NewMenu(items ...string) Menu {
	return Menu{items: items}
}

func (m *Menu) Update(key string) (int, bool) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "enter":
		return m.cursor, true
	}
	return 0, false
}

func (m Menu) View() string {
	var b strings.Builder
	for i, item := range m.items {
		if i == m.cursor {
			b.WriteString(styles.Selected.Render("> " + item))
		} else {
			b.WriteString("  ")
			b.WriteString(item)
		}
		b.WriteString("\n")
	}
	return b.String()
}
