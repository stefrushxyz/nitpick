package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v57/github"
	"github.com/stefrushxyz/nitpick/internal/clipboard"
	ghclient "github.com/stefrushxyz/nitpick/internal/github"
	"github.com/stefrushxyz/nitpick/internal/prompt"
	"github.com/stefrushxyz/nitpick/internal/ui"
)

// State represents the current view state
type State int

const (
	StateRepos State = iota
	StatePRs
	StateComments
	StateCommentDetail
)

// App represents the main application
type App struct {
	client          *ghclient.Client
	promptGen       *prompt.Generator
	state           State
	repoList        list.Model
	prList          list.Model
	commentList     list.Model
	commentViewport viewport.Model
	currentRepo     *github.Repository
	currentPR       *github.PullRequest
	currentComment  *github.PullRequestComment
	loading         bool
	err             error
	width           int
	height          int
	copyStatus      string // Status message for copy operations
	showReplies     bool   // Whether to show reply comments
	useSimplePrompt bool   // Whether to use simple prompt template
}

// New creates a new application instance
func New(token string) *App {
	// Create GitHub client
	client := ghclient.New(token)

	// Create prompt generator
	promptGen := prompt.New()

	// Initialize lists
	repoList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	repoList.Title = "GitHub Repositories"
	repoList.Styles.TitleBar.PaddingLeft(0)
	repoList.SetShowStatusBar(false)
	repoList.SetFilteringEnabled(true)

	prList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	prList.Title = "Pull Requests"
	prList.Styles.TitleBar.PaddingLeft(0)
	prList.SetShowStatusBar(false)
	prList.SetFilteringEnabled(true)

	commentList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	commentList.Title = "PR Comments"
	commentList.Styles.TitleBar.PaddingLeft(0)
	commentList.SetShowStatusBar(false)
	commentList.SetFilteringEnabled(true)

	// Initialize viewport for comment details
	commentViewport := viewport.New(0, 0)

	return &App{
		client:          client,
		promptGen:       promptGen,
		state:           StateRepos,
		repoList:        repoList,
		prList:          prList,
		commentList:     commentList,
		commentViewport: commentViewport,
		loading:         true,
		showReplies:     false,
		useSimplePrompt: false,
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.fetchRepos(),
		tea.EnterAltScreen,
	)
}

// Update handles messages and updates the application state
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.repoList.SetSize(msg.Width-4, msg.Height-4)
		a.prList.SetSize(msg.Width-4, msg.Height-4)
		a.commentList.SetSize(msg.Width-4, msg.Height-7)

		availableHeight := msg.Height - 5
		if a.copyStatus != "" {
			availableHeight -= 2
		}
		a.commentViewport.Width = msg.Width - 4
		a.commentViewport.Height = availableHeight

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "esc":
			switch a.state {
			case StateRepos:
				if a.repoList.SettingFilter() {
					var cmd tea.Cmd
					a.repoList, cmd = a.repoList.Update(msg)
					return a, cmd
				}
			case StatePRs:
				if a.prList.SettingFilter() {
					var cmd tea.Cmd
					a.prList, cmd = a.prList.Update(msg)
					return a, cmd
				}
			case StateComments:
				if a.commentList.SettingFilter() {
					var cmd tea.Cmd
					a.commentList, cmd = a.commentList.Update(msg)
					return a, cmd
				}
			}
			return a.handleBack()
		case "enter":
			return a.handleEnter()
		case "c":
			if a.state == StateCommentDetail {
				return a.handleCopyPrompt()
			}
		case "t":
			if a.state == StateCommentDetail {
				return a.handleTogglePromptMode()
			}
		case "r":
			if a.state == StateComments {
				return a.handleToggleReplies()
			}
		case "up", "k":
			if a.state == StateCommentDetail {
				a.commentViewport.LineUp(1)
				return a, nil
			}
		case "down", "j":
			if a.state == StateCommentDetail {
				a.commentViewport.LineDown(1)
				return a, nil
			}
		case "pgup", "h":
			if a.state == StateCommentDetail {
				a.commentViewport.HalfViewUp()
				return a, nil
			}
		case "pgdown", "l":
			if a.state == StateCommentDetail {
				a.commentViewport.HalfViewDown()
				return a, nil
			}
		case "home", "g":
			if a.state == StateCommentDetail {
				a.commentViewport.GotoTop()
				return a, nil
			}
		case "end", "G":
			if a.state == StateCommentDetail {
				a.commentViewport.GotoBottom()
				return a, nil
			}
		}

	case ghclient.ReposMsg:
		a.loading = false
		if msg.Err != nil {
			a.err = msg.Err
			return a, nil
		}
		items := make([]list.Item, len(msg.Repos))
		for i, repo := range msg.Repos {
			items[i] = ui.RepoItem{Repo: repo}
		}
		a.repoList.SetItems(items)

	case ghclient.PRsMsg:
		a.loading = false
		if msg.Err != nil {
			a.err = msg.Err
			return a, nil
		}
		items := make([]list.Item, len(msg.PRs))
		for i, pr := range msg.PRs {
			items[i] = ui.PRItem{PR: pr}
		}
		a.prList.SetItems(items)

	case ghclient.CommentsMsg:
		a.loading = false
		if msg.Err != nil {
			a.err = msg.Err
			return a, nil
		}

		// Filter comments based on showReplies setting
		var filteredComments []*github.PullRequestComment
		for _, comment := range msg.Comments {
			if a.showReplies || comment.GetInReplyTo() == 0 {
				filteredComments = append(filteredComments, comment)
			}
		}

		items := make([]list.Item, len(filteredComments))
		for i, comment := range filteredComments {
			items[i] = ui.CommentItem{Comment: comment}
		}
		a.commentList.SetItems(items)

	case clearCopyStatusMsg:
		a.copyStatus = ""
	}

	// Update the current list or viewport
	var cmd tea.Cmd
	switch a.state {
	case StateRepos:
		a.repoList, cmd = a.repoList.Update(msg)
	case StatePRs:
		a.prList, cmd = a.prList.Update(msg)
	case StateComments:
		a.commentList, cmd = a.commentList.Update(msg)
	case StateCommentDetail:
		a.commentViewport, cmd = a.commentViewport.Update(msg)
	}

	return a, cmd
}

// View renders the application
func (a *App) View() string {
	if a.loading {
		return lipgloss.NewStyle().
			Align(lipgloss.Center).
			Render("Loading...")
	}

	if a.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render(fmt.Sprintf("Error: %v", a.err))
	}

	var content string
	var breadcrumb string

	switch a.state {
	case StateRepos:
		content = a.repoList.View()
		breadcrumb = "Repositories"
	case StatePRs:
		content = a.prList.View()
		breadcrumb = fmt.Sprintf("Repositories > %s > Pull Requests", a.currentRepo.GetName())
	case StateComments:
		prInfo := a.buildPRInfo()
		content = lipgloss.JoinVertical(lipgloss.Left,
			prInfo,
			a.commentList.View(),
		)
		breadcrumb = fmt.Sprintf("Repositories > %s > Pull Requests > #%d > Comments",
			a.currentRepo.GetName(), a.currentPR.GetNumber())
	case StateCommentDetail:
		content = a.commentViewport.View()
		breadcrumb = fmt.Sprintf("Repositories > %s > Pull Requests > #%d > Comments > Comment",
			a.currentRepo.GetName(), a.currentPR.GetNumber())
	}

	// Build help text based on current state
	var helpText string
	if a.state == StateCommentDetail {
		promptMode := "full"
		if a.useSimplePrompt {
			promptMode = "simple"
		}
		helpText = fmt.Sprintf("c: copy prompt (%s) • t: toggle prompt mode • ↑/↓ j/k: scroll • Esc: back • q: quit", promptMode)
	} else if a.state == StateComments {
		repliesStatus := "show"
		if a.showReplies {
			repliesStatus = "hide"
		}
		helpText = fmt.Sprintf("Enter: select • r: %s replies • Esc: back • q: quit", repliesStatus)
	} else {
		helpText = "Enter: select • Esc: back • q: quit"
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(helpText)

	if a.state == StateCommentDetail {
		// Calculate viewport height
		fixedLines := 6
		viewportHeight := max(a.height-fixedLines, 1)

		// Update viewport size if needed
		if a.commentViewport.Height != viewportHeight {
			a.commentViewport.Height = viewportHeight
		}

		// Build header elements (just breadcrumb, no status here)
		header := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render(breadcrumb),
			"",
		)

		// Build status section (below viewport)
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)
		statusSection := statusStyle.Render(a.copyStatus)

		// Build final layout with status below viewport
		var layoutElements []string
		layoutElements = append(layoutElements, header)
		layoutElements = append(layoutElements, content)

		if statusSection != "" {
			layoutElements = append(layoutElements, "", statusSection)
		} else {
			layoutElements = append(layoutElements, "")
		}

		layoutElements = append(layoutElements, "", help)

		return lipgloss.JoinVertical(lipgloss.Left, layoutElements...)
	}

	// Original layout for other states
	var elements []string
	elements = append(elements, lipgloss.NewStyle().Bold(true).Render(breadcrumb))
	elements = append(elements, "")

	// Add status if present
	if a.copyStatus != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)
		elements = append(elements, statusStyle.Render(a.copyStatus))
		elements = append(elements, "")
	}

	elements = append(elements, content)
	elements = append(elements, "")
	elements = append(elements, help)

	return lipgloss.JoinVertical(lipgloss.Left, elements...)
}

// handleEnter handles the enter key press
func (a *App) handleEnter() (tea.Model, tea.Cmd) {
	switch a.state {
	case StateRepos:
		selected := a.repoList.SelectedItem()
		if selected != nil {
			item := selected.(ui.RepoItem)
			a.currentRepo = item.Repo
			a.state = StatePRs
			a.loading = true
			return a, a.fetchPRs()
		}
	case StatePRs:
		selected := a.prList.SelectedItem()
		if selected != nil {
			item := selected.(ui.PRItem)
			a.currentPR = item.PR
			a.state = StateComments
			a.loading = true
			return a, a.fetchComments()
		}
	case StateComments:
		selected := a.commentList.SelectedItem()
		if selected != nil {
			item := selected.(ui.CommentItem)
			a.currentComment = item.Comment
			a.state = StateCommentDetail

			// Calculate proper viewport height before setting content
			// Use same logic as View method: fixed 6 lines for UI elements
			fixedLines := 6
			viewportHeight := max(a.height-fixedLines, 1)

			// Set viewport dimensions
			a.commentViewport.Width = a.width - 4
			a.commentViewport.Height = viewportHeight

			// Set up viewport content
			content := a.buildCommentDetail()
			a.commentViewport.SetContent(content)

			return a, nil
		}
	}
	return a, nil
}

// handleBack handles the back navigation
func (a *App) handleBack() (tea.Model, tea.Cmd) {
	switch a.state {
	case StatePRs:
		a.state = StateRepos
		a.currentRepo = nil
	case StateComments:
		a.state = StatePRs
		a.currentPR = nil
	case StateCommentDetail:
		a.state = StateComments
		a.currentComment = nil
	}
	return a, nil
}

// fetchRepos fetches repositories from GitHub
func (a *App) fetchRepos() tea.Cmd {
	return a.client.FetchRepos()
}

// fetchPRs fetches pull requests for the current repository
func (a *App) fetchPRs() tea.Cmd {
	if a.currentRepo == nil {
		return nil
	}
	return a.client.FetchPRs(a.currentRepo)
}

// fetchComments fetches comments for the current pull request
func (a *App) fetchComments() tea.Cmd {
	if a.currentRepo == nil || a.currentPR == nil {
		return nil
	}
	return a.client.FetchComments(a.currentRepo, a.currentPR)
}

// handleCopyPrompt handles copying the prompt to clipboard based on current mode
func (a *App) handleCopyPrompt() (tea.Model, tea.Cmd) {
	if a.currentRepo == nil || a.currentPR == nil || a.currentComment == nil {
		a.copyStatus = "Error: Missing context for prompt generation"
		return a, nil
	}

	// Generate prompt based on current mode
	var promptText string
	var promptType string
	if a.useSimplePrompt {
		promptText = a.promptGen.GenerateSimplePrompt(a.currentRepo, a.currentPR, a.currentComment)
		promptType = "Simple"
	} else {
		promptText = a.promptGen.GenerateFullPrompt(a.currentRepo, a.currentPR, a.currentComment)
		promptType = "Full"
	}

	// Copy to clipboard
	if err := clipboard.Copy(promptText); err != nil {
		a.copyStatus = fmt.Sprintf("Copy failed: %v", err)
	} else {
		a.copyStatus = fmt.Sprintf("✅ %s prompt copied to clipboard!", promptType)
	}

	// Clear status after 3 seconds
	return a, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return clearCopyStatusMsg{}
	})
}

// handleTogglePromptMode toggles between simple and full prompt modes
func (a *App) handleTogglePromptMode() (tea.Model, tea.Cmd) {
	a.useSimplePrompt = !a.useSimplePrompt

	mode := "Full"
	if a.useSimplePrompt {
		mode = "Simple"
	}

	a.copyStatus = fmt.Sprintf("🔄 Switched to %s prompt mode", mode)

	// Clear status after 2 seconds
	return a, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return clearCopyStatusMsg{}
	})
}

// handleToggleReplies toggles the showReplies setting and refetches comments
func (a *App) handleToggleReplies() (tea.Model, tea.Cmd) {
	a.showReplies = !a.showReplies
	a.loading = true
	return a, a.fetchComments()
}

// clearCopyStatusMsg is used to clear the copy status message
type clearCopyStatusMsg struct{}

// buildPRInfo creates a formatted display of PR information
func (a *App) buildPRInfo() string {
	if a.currentPR == nil {
		return ""
	}
	return a.buildSelectedPRInfo(a.currentPR)
}

// buildSelectedPRInfo creates a formatted display of PR information for any given PR
func (a *App) buildSelectedPRInfo(pr *github.PullRequest) string {
	if pr == nil {
		return ""
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginBottom(1)

	// PR title
	title := fmt.Sprintf("#%d %s", pr.GetNumber(), pr.GetTitle())

	// PR metadata
	author := pr.GetUser().GetLogin()
	created := ""
	if pr.CreatedAt != nil {
		created = pr.CreatedAt.Format("2006-01-02 15:04")
	}

	var statusParts []string
	if pr.GetDraft() {
		statusParts = append(statusParts, "DRAFT")
	}
	if pr.GetMerged() {
		statusParts = append(statusParts, "MERGED")
	}

	state := pr.GetState()
	switch state {
	case "open":
		statusParts = append(statusParts, "🟢 OPEN")
	case "closed":
		statusParts = append(statusParts, "🔴 CLOSED")
	}

	statusStr := ""
	if len(statusParts) > 0 {
		statusStr = fmt.Sprintf(" [%s]", strings.Join(statusParts, ", "))
	}

	meta := fmt.Sprintf("by %s • %s%s", author, created, statusStr)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		metaStyle.Render(meta),
	)
}

// buildCommentDetail creates a formatted display of comment detail
func (a *App) buildCommentDetail() string {
	if a.currentComment == nil {
		return ""
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1).
		MarginBottom(1)

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248")).
		MarginBottom(1)

	var sections []string

	// Main Title with PR Name
	title := fmt.Sprintf("Comment on #%d %s", a.currentPR.GetNumber(), a.currentPR.GetTitle())
	prTitle := titleStyle.Render(title)
	sections = append(sections, prTitle)

	// Comment author and metadata
	author := a.currentComment.User.GetLogin()
	created := ""
	if a.currentComment.CreatedAt != nil {
		created = a.currentComment.CreatedAt.Format("2006-01-02 15:04")
	}
	updated := ""
	if a.currentComment.UpdatedAt != nil && !a.currentComment.UpdatedAt.Equal(*a.currentComment.CreatedAt) {
		updated = fmt.Sprintf(" (updated %s)", a.currentComment.UpdatedAt.Format("2006-01-02 15:04"))
	}

	commentMeta := fmt.Sprintf("By: %s\nCreated: %s%s", author, created, updated)
	sections = append(sections, metaStyle.Render(commentMeta))

	// Comment body with markdown rendering
	body := a.currentComment.GetBody()
	if body == "" {
		body = "No content provided"
	}

	// Render markdown content with enhanced Glamour styling
	rendered, err := a.renderMarkdown(body)
	if err != nil {
		// Fallback to styled plain text if markdown rendering fails
		fallbackStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("242")).
			MarginBottom(1)
		sections = append(sections, fallbackStyle.Render(body))
	} else {
		sections = append(sections, rendered)
	}

	sections = append(sections, "")

	// Code Context Section
	if a.currentComment.GetPath() != "" || a.currentComment.GetDiffHunk() != "" {
		// File and line information
		if a.currentComment.GetPath() != "" {
			fileContext := a.renderFileContext(a.currentComment.GetPath(),
				a.currentComment.GetLine(),
				a.currentComment.GetOriginalLine())
			sections = append(sections, fileContext)
		}

		// Code diff context
		if a.currentComment.GetDiffHunk() != "" {
			codeContext := a.renderCodeContext(a.currentComment.GetDiffHunk())
			sections = append(sections, codeContext)
		} else {
			sections = append(sections, "")
		}
	}

	// Direct Link Section
	if a.currentComment.GetHTMLURL() != "" {
		directLink := a.renderDirectLink(a.currentComment.GetHTMLURL())
		sections = append(sections, directLink)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderMarkdown renders markdown content using Glamour with terminal-appropriate styling
func (a *App) renderMarkdown(content string) (string, error) {
	// Determine word wrap width with sensible defaults
	wrapWidth := 80 // Default width
	if a.width > 16 {
		wrapWidth = a.width - 8 // Account for padding
	}

	// Create a renderer with enhanced terminal-friendly styling
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrapWidth),
		glamour.WithStylePath("dark"),
	)
	if err != nil {
		return "", err
	}

	// Render the markdown
	rendered, err := renderer.Render(content)
	if err != nil {
		return "", err
	}

	// Remove trailing whitespace that Glamour sometimes adds
	return strings.TrimSpace(rendered), nil
}

// renderCodeContext creates an enhanced, well-formatted display of diff/code context
func (a *App) renderCodeContext(diffHunk string) string {
	if diffHunk == "" {
		return ""
	}

	// Enhanced styling for code context
	codeBlockStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("234")).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		MarginBottom(1)

	// Try to render the diff as markdown for syntax highlighting
	diffMarkdown := "```diff\n" + diffHunk + "\n```"
	rendered, err := a.renderMarkdown(diffMarkdown)
	if err != nil {
		// Fallback to plain styling if markdown rendering fails
		return codeBlockStyle.Render(diffHunk)
	}

	return rendered
}

// renderFileContext creates an enhanced display of file and line information
func (a *App) renderFileContext(path string, line int, originalLine int) string {
	if path == "" {
		return ""
	}

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248"))

	var info []string
	info = append(info, fmt.Sprintf("📁 %s", path))

	// Add line information if available
	if line != 0 {
		// Check if this is a multi-line comment (has start_line)
		startLine := a.currentComment.GetStartLine()
		if startLine != 0 && startLine != line {
			info = append(info, fmt.Sprintf("📍 Lines: L%d-%d", startLine, line))
		} else {
			info = append(info, fmt.Sprintf("📍 Line: L%d", line))
		}
	}
	if originalLine != 0 && originalLine != line {
		// Check for original start line for multi-line comments
		originalStartLine := a.currentComment.GetOriginalStartLine()
		if originalStartLine != 0 && originalStartLine != originalLine {
			info = append(info, fmt.Sprintf("📍 Original Lines: L%d-%d", originalStartLine, originalLine))
		} else {
			info = append(info, fmt.Sprintf("📍 Original Line: L%d", originalLine))
		}
	}

	return infoStyle.Render(strings.Join(info, " • "))
}

// renderDirectLink creates an enhanced, actionable display of the direct link
func (a *App) renderDirectLink(url string) string {
	if url == "" {
		return ""
	}

	linkContentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("110")).
		Background(lipgloss.Color("235")).
		Padding(1, 2).
		MarginBottom(1).
		Underline(true)

	linkInstructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248")).
		Italic(true).
		MarginBottom(1)

	instruction := "💡 Copy this URL to open the comment directly in your browser"

	return lipgloss.JoinVertical(lipgloss.Left,
		linkContentStyle.Render(url),
		linkInstructionStyle.Render(instruction),
	)
}
