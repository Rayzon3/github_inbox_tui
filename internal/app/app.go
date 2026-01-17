package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// NewProgramModel constructs the Bubble Tea model for the app.
func NewProgramModel() tea.Model {
	styles := newStyles()
	listModel := initList()
	return newModel(listModel, styles)
}
