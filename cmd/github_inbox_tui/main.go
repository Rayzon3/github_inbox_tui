package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github_inbox_tui/internal/app"
)

func main() {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		stored, err := loadToken()
		if err == nil {
			token = stored
		}
	}
	if token == "" {
		entered, err := promptToken()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		token = entered
		if err := saveToken(token); err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not save token:", err.Error())
		}
	}
	_ = os.Setenv("GITHUB_TOKEN", token)

	m := app.NewProgramModel()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
