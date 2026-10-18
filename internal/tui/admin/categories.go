package admin

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/tui/components"
)

type categoriesLoadedMsg struct {
	categories []dto.CategoryResponse
	err        error
}

// categoryChangedMsg arrive après une création, un renommage ou une suppression.
type categoryChangedMsg struct {
	info string
	err  error
}

func loadCategoriesCmd(api *apiclient.Client) tea.Cmd {
	return func() tea.Msg {
		categories, err := api.ListCategories(context.Background())
		return categoriesLoadedMsg{categories: categories, err: err}
	}
}

func createCategoryCmd(api *apiclient.Client, name string) tea.Cmd {
	return func() tea.Msg {
		err := api.CreateCategory(context.Background(), name)
		return categoryChangedMsg{info: "Catégorie créée.", err: err}
	}
}

func renameCategoryCmd(api *apiclient.Client, id int, name string) tea.Cmd {
	return func() tea.Msg {
		err := api.RenameCategory(context.Background(), id, name)
		return categoryChangedMsg{info: "Catégorie renommée.", err: err}
	}
}

func deleteCategoryCmd(api *apiclient.Client, id int) tea.Cmd {
	return func() tea.Msg {
		err := api.DeleteCategory(context.Background(), id)
		return categoryChangedMsg{info: "Catégorie supprimée.", err: err}
	}
}

// goCategories affiche la liste : « Nouvelle catégorie », les catégories, puis « Retour ».
func (m *Model) goCategories() {
	items := []string{"+ Nouvelle catégorie"}
	for _, c := range m.categories {
		items = append(items, c.Name)
	}
	items = append(items, "← Retour")

	m.screen = screenCategories
	m.menu = components.NewMenu(items...)
}

// Même ordre que dans goCategoryActions.
const (
	actionRename = iota
	actionDelete
	actionBack
)

func (m *Model) goCategoryActions() {
	m.screen = screenCategoryActions
	m.menu = components.NewMenu("Renommer", "Supprimer", "← Retour")
	m.err, m.info = "", ""
}

// goCategoryForm ouvre le formulaire. Avec selected.ID == 0, c'est une création.
func (m *Model) goCategoryForm() {
	m.screen = screenCategoryForm
	m.form = components.NewForm(
		components.Field{Label: "Nom", Placeholder: "ex. jardin", Value: m.selected.Name},
	)
	m.err, m.info = "", ""
}

func (m Model) updateCategories(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}

	switch {
	case choice == 0: // + Nouvelle catégorie
		m.selected = dto.CategoryResponse{}
		m.goCategoryForm()
	case choice == len(m.categories)+1: // ← Retour
		m.goMain()
	default:
		m.selected = m.categories[choice-1] // -1 : la 1re ligne est « Nouvelle catégorie »
		m.goCategoryActions()
	}
	return m, nil
}

func (m Model) updateCategoryActions(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}

	switch choice {
	case actionRename:
		m.goCategoryForm()
	case actionDelete:
		m.loading = true
		return m, deleteCategoryCmd(m.api, m.selected.ID)
	case actionBack:
		m.goCategories()
	}
	return m, nil
}

func (m Model) updateCategoryForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	submitted, cmd := m.form.Update(msg)
	if !submitted {
		return m, cmd
	}

	name := m.form.Value(0)
	m.err, m.info = "", ""
	m.loading = true
	if m.selected.ID == 0 {
		return m, createCategoryCmd(m.api, name)
	}
	return m, renameCategoryCmd(m.api, m.selected.ID, name)
}
