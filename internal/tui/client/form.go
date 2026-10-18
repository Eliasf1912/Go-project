package client

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type field struct {
	label       string
	placeholder string
	secret      bool
	value       string
}

// Sans largeur, bubbles n'affiche que la première lettre du placeholder.
const inputWidth = 40

type form struct {
	inputs []textinput.Model
	focus  int
}

func newForm(descriptions ...field) form {
	f := form{inputs: make([]textinput.Model, len(descriptions))}
	firstEmpty := -1

	labelWidth := 0
	for _, desc := range descriptions {
		labelWidth = max(labelWidth, len([]rune(desc.label)))
	}

	for index, desc := range descriptions {
		input := textinput.New()
		input.Prompt = fmt.Sprintf("%-*s : ", labelWidth, desc.label)
		input.Placeholder = desc.placeholder
		input.SetWidth(inputWidth)
		input.SetValue(desc.value)
		if desc.secret {
			input.EchoMode = textinput.EchoPassword
			input.EchoCharacter = '•'
		}
		f.inputs[index] = input

		if desc.value == "" && firstEmpty == -1 {
			firstEmpty = index
		}
	}

	if firstEmpty == -1 {
		firstEmpty = 0
	}
	f.setFocus(firstEmpty)
	return f
}

func (f *form) setFocus(target int) tea.Cmd {
	total := len(f.inputs)
	f.focus = (target + total) % total // revient au début après le dernier, et inversement

	var cmd tea.Cmd
	for index := range f.inputs {
		if index == f.focus {
			cmd = f.inputs[index].Focus()
		} else {
			f.inputs[index].Blur()
		}
	}
	return cmd
}

func (f *form) update(msg tea.Msg) (bool, tea.Cmd) {
	if key, isKey := msg.(tea.KeyPressMsg); isKey {
		switch key.String() {
		case "tab", "down":
			return false, f.setFocus(f.focus + 1)
		case "shift+tab", "up":
			return false, f.setFocus(f.focus - 1)
		case "enter":
			isLastField := f.focus == len(f.inputs)-1
			if isLastField {
				return true, nil
			}
			return false, f.setFocus(f.focus + 1)
		}
	}

	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return false, cmd
}

func (f form) value(index int) string {
	return f.inputs[index].Value()
}

func (f form) view() string {
	var builder strings.Builder
	for _, input := range f.inputs {
		builder.WriteString(input.View())
		builder.WriteString("\n")
	}
	return builder.String()
}
