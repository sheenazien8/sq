package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sheenazien8/sq/app"
	"github.com/sheenazien8/sq/internal/version"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/storage"
)

func main() {
	// Parse command line flags
	versionFlag := flag.Bool("version", false, "Show version information")
	versionShort := flag.Bool("v", false, "Show version information (short)")

	// Connection creation flags
	createConnFlag := flag.Bool("create-connection", false, "Create a new database connection")
	connDriver := flag.String("driver", "mysql", "Database driver (mysql)")
	connName := flag.String("name", "", "Connection name")
	connHost := flag.String("host", "localhost", "Database host")
	connPort := flag.String("port", "3306", "Database port")
	connUser := flag.String("user", "", "Database user")
	connPass := flag.String("password", "", "Database password")
	connDB := flag.String("database", "", "Database name")

	flag.Parse()

	// Handle version flag
	if *versionFlag || *versionShort {
		fmt.Printf("sq version %s\n", version.Version)
		os.Exit(0)
	}

	// Handle create connection flag
	if *createConnFlag {
		if err := handleCreateConnection(*connDriver, *connName, *connHost, *connPort, *connUser, *connPass, *connDB); err != nil {
			fmt.Printf("Error creating connection: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Connection created successfully!")
		os.Exit(0)
	}

	// Setup logger
	if err := logger.SetFile("debug.log"); err != nil {
		fmt.Println("Failed to setup logger:", err)
		os.Exit(1)
	}

	// Set log level based on DEBUG environment variable
	if os.Getenv("DEBUG") == "true" {
		logger.SetLevel(slog.LevelDebug)
	} else {
		logger.SetLevel(slog.LevelInfo)
	}
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

// handleCreateConnection creates a new database connection from CLI flags
func handleCreateConnection(driver, name, host, port, user, password, database string) error {
	// Validate driver
	if driver != "mysql" {
		return fmt.Errorf("unsupported driver: %s (supported: mysql)", driver)
	}

	// Validate required fields
	if name == "" {
		return fmt.Errorf("connection name is required (--name)")
	}
	if user == "" {
		return fmt.Errorf("database user is required (--user)")
	}
	if database == "" {
		return fmt.Errorf("database name is required (--database)")
	}

	// Initialize storage
	if err := storage.Init(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer storage.Close()

	// Setup logger (minimal for CLI usage)
	if err := logger.SetFile("debug.log"); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	// Build MySQL connection URL
	var url string
	if password == "" {
		url = fmt.Sprintf("mysql://%s@%s:%s/%s", user, host, port, database)
	} else {
		url = fmt.Sprintf("mysql://%s:%s@%s:%s/%s", user, password, host, port, database)
	}

	// Create connection (this will test the connection before saving)
	_, err := storage.CreateConnection(name, driver, url)
	if err != nil {
		return err
	}

	return nil
}
