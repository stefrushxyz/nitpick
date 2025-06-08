package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v57/github"
)

// RepoItem represents a repository in the list
type RepoItem struct {
	Repo *github.Repository
}

// FilterValue returns the name of a repository
func (i RepoItem) FilterValue() string {
	return i.Repo.GetName()
}

// Title returns the title of a repository
func (i RepoItem) Title() string {
	name := i.Repo.GetName()
	if i.Repo.GetOwner().GetLogin() != "" {
		name = fmt.Sprintf("%s/%s", i.Repo.GetOwner().GetLogin(), name)
	}

	// Add indicators for private repos and forks
	var indicators []string
	if i.Repo.GetPrivate() {
		indicators = append(indicators, "🔒")
	}
	if i.Repo.GetFork() {
		indicators = append(indicators, "🍴")
	}

	if len(indicators) > 0 {
		name = fmt.Sprintf("%s %s", name, strings.Join(indicators, " "))
	}

	return name
}

// Description returns the description of a repository
func (i RepoItem) Description() string {
	desc := i.Repo.GetDescription()
	if desc == "" {
		desc = "No description"
	}

	// Add language and last updated info
	lang := i.Repo.GetLanguage()
	updated := ""
	if i.Repo.UpdatedAt != nil {
		updated = i.Repo.UpdatedAt.Format("2006-01-02")
	}

	if lang != "" && updated != "" {
		return fmt.Sprintf("%s • %s • Updated %s", desc, lang, updated)
	} else if lang != "" {
		return fmt.Sprintf("%s • %s", desc, lang)
	} else if updated != "" {
		return fmt.Sprintf("%s • Updated %s", desc, updated)
	}

	return desc
}

// PRItem represents a pull request in the list
type PRItem struct {
	PR *github.PullRequest
}

// FilterValue returns the title of a pull request
func (i PRItem) FilterValue() string {
	return i.PR.GetTitle()
}

// Title returns the title of a pull request
func (i PRItem) Title() string {
	return fmt.Sprintf("#%d %s", i.PR.GetNumber(), i.PR.GetTitle())
}

// Description returns the description of a pull request
func (i PRItem) Description() string {
	author := i.PR.GetUser().GetLogin()
	created := ""
	if i.PR.CreatedAt != nil {
		created = i.PR.CreatedAt.Format("2006-01-02")
	}

	// Add status indicators
	var status []string
	if i.PR.GetDraft() {
		status = append(status, "DRAFT")
	}
	if i.PR.GetMerged() {
		status = append(status, "MERGED")
	}

	statusStr := ""
	if len(status) > 0 {
		statusStr = fmt.Sprintf("[%s] ", strings.Join(status, ", "))
	}

	return fmt.Sprintf("%sby %s • Created %s", statusStr, author, created)
}

// CommentItem represents a PR comment in the list
type CommentItem struct {
	Comment *github.PullRequestComment
}

// FilterValue returns the body of a comment
func (i CommentItem) FilterValue() string {
	return i.Comment.GetBody()
}

// Title returns the title of a comment
func (i CommentItem) Title() string {
	body := strings.TrimSpace(i.Comment.GetBody())
	lines := strings.Split(body, "\n")

	// Take first non-empty line as title
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Strip Markdown title prefix if present
			re := regexp.MustCompile(`^\s*#\s*`)
			line = re.ReplaceAllString(line, "")

			// Truncate if too long
			if len(line) > 80 {
				line = line[:77] + "..."
			}
			return line
		}
	}

	return "Empty comment"
}

// Description returns the description of a comment
func (i CommentItem) Description() string {
	author := i.Comment.GetUser().GetLogin()
	created := ""
	if i.Comment.CreatedAt != nil {
		created = i.Comment.CreatedAt.Format("2006-01-02 15:04")
	}

	// Show updated time if different from created time
	timeInfo := created
	if i.Comment.UpdatedAt != nil && i.Comment.CreatedAt != nil {
		updated := i.Comment.UpdatedAt.Format("2006-01-02 15:04")
		if updated != created {
			timeInfo = fmt.Sprintf("%s (updated %s)", created, updated)
		}
	}

	// Build file and line information using the same logic as detail view
	fileInfo := ""
	if i.Comment.GetPath() != "" {
		fileInfo = fmt.Sprintf(" • %s", i.Comment.GetPath())

		// Add line information - handle multi-line comments properly
		line := i.Comment.GetLine()
		startLine := i.Comment.GetStartLine()
		originalLine := i.Comment.GetOriginalLine()
		originalStartLine := i.Comment.GetOriginalStartLine()

		// Current line information
		if line != 0 {
			if startLine != 0 && startLine != line {
				// Multi-line comment (has start_line)
				fileInfo += fmt.Sprintf(" L%d-%d", startLine, line)
			} else {
				// Single line comment
				fileInfo += fmt.Sprintf(" L%d", line)
			}
		}

		// Original line information (if different from current)
		if originalLine != 0 && originalLine != line {
			if originalStartLine != 0 && originalStartLine != originalLine {
				// Multi-line original comment
				fileInfo += fmt.Sprintf(" (orig L%d-%d)", originalStartLine, originalLine)
			} else {
				// Single line original comment
				fileInfo += fmt.Sprintf(" (orig L%d)", originalLine)
			}
		}
	}

	return fmt.Sprintf("by %s • %s%s", author, timeInfo, fileInfo)
}
