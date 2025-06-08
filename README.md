# Nitpick

A terminal-based GitHub browser that helps you generate AI prompts to implement code changes in your pull requests.

## Features

- Browse your GitHub repositories in a clean TUI interface
- View pull requests and their details
- Read and navigate PR comments (with reply filtering)
- Generate AI-friendly prompts from PR context for code review
- Copy prompts to clipboard for use with AI tools
- Built with Go and the Bubble Tea framework

## Prerequisites

- Go 1.23 or later
- A GitHub Personal Access Token

## Setup

1. **Clone the repository:**

   ```bash
   git clone https://github.com/stefrushxyz/nitpick.git
   cd nitpick
   ```

2. **Install dependencies:**

   ```bash
   make deps
   ```

3. **Create a GitHub Personal Access Token:**

   - Go to [GitHub Settings > Tokens](https://github.com/settings/personal-access-tokens)
   - Create a new token with appropriate permissions for reading repositories and pull requests

4. **Set up your GitHub token:**
   
   Either set an environment variable:
   ```bash
   export GITHUB_TOKEN=your_personal_access_token
   ```
   
   Or create a `.env` file in the project root:

   ```
   GITHUB_TOKEN=your_personal_access_token
   ```

## Usage

### Running the Application

```bash
# Run directly
make run

# Or build and run
make build
./bin/nitpick
```

### Navigation Commands

- **Arrow keys or j/k**: Navigate through lists
- **Enter**: Select item/drill down
- **Esc**: Go back to previous view
- **q or Ctrl+C**: Quit application

### Comment View Commands

- **c**: Copy AI prompt to clipboard
- **p**: Toggle between simple and full prompt modes
- **r**: Toggle reply comments visibility (in comments list)
- **Arrow keys/j/k**: Scroll through comment content
- **Page Up/Down**: Scroll by half-page

## Building

```bash
# Build for current platform
make build

# Build for multiple platforms
make build-all

# Clean build artifacts
make clean
```

## Project Structure

```
├── cmd/nitpick/          # Main application entry point
├── internal/
│   ├── app/              # Core application logic and TUI
│   ├── clipboard/        # Clipboard operations
│   ├── github/           # GitHub API client
│   ├── prompt/           # AI prompt generation
│   └── ui/               # UI components
├── bin/                  # Built binaries
└── Makefile              # Build and development commands
```

## Development

The application uses:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) for reusable components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling
- [Glamour](https://github.com/charmbracelet/glamour) for Markdown rendering
- [GitHub API v4](https://github.com/google/go-github) for GitHub integration
