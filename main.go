package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sheenazien8/db-client-tui/app"
)

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(
		app.New(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
