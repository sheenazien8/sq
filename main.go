package main

import (
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sheenazien8/db-client-tui/app"
	"github.com/sheenazien8/db-client-tui/logger"
	"github.com/sheenazien8/db-client-tui/storage"
)

func main() {
	// Setup logger
	if err := logger.SetFile("debug.log"); err != nil {
		fmt.Println("Failed to setup logger:", err)
		os.Exit(1)
	}
	logger.SetLevel(slog.LevelDebug)
	logger.Info("Application started", nil)

	// Initialize app storage (SQLite database)
	if err := storage.Init(); err != nil {
		logger.Error("Failed to initialize storage", map[string]any{"error": err.Error()})
		fmt.Println("Failed to initialize storage:", err)
		os.Exit(1)
	}
	defer storage.Close()

	p := tea.NewProgram(
		app.New(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
