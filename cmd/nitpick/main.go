package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/stefrushxyz/nitpick/internal/app"
)

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Check for GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Please set GITHUB_TOKEN environment variable")
		fmt.Println("You can either:")
		fmt.Println("  1. Set environment variable: export GITHUB_TOKEN=your_token")
		fmt.Println("  2. Create a .env file with: GITHUB_TOKEN=your_token")
		fmt.Println("You can create a personal access token at: https://github.com/settings/personal-access-tokens")
		os.Exit(1)
	}

	// Initialize the TUI application
	application := app.New(token)
	p := tea.NewProgram(application, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
