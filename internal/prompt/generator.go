package prompt

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/google/go-github/v57/github"
)

// Generator handles creating prompts for GitHub Copilot
type Generator struct {
	fullTemplate   *template.Template
	simpleTemplate *template.Template
}

// TemplateData holds all the data needed for prompt generation
type TemplateData struct {
	Repository  *RepositoryData
	PullRequest *PullRequestData
	Comment     *CommentData
	Generated   string
}

type RepositoryData struct {
	FullName    string
	Name        string
	Description string
	Language    string
}

type PullRequestData struct {
	Number       int
	Title        string
	Author       string
	State        string
	IsDraft      bool
	IsMerged     bool
	Created      string
	Body         string
	SourceBranch string
	TargetBranch string
}

type CommentData struct {
	Reviewer          string
	Date              string
	Path              string
	Line              int
	StartLine         int
	OriginalLine      int
	OriginalStartLine int
	LineRange         string
	OriginalLineRange string
	DiffHunk          string
	Body              string
	HTMLURL           string
}

const fullPromptTemplate = `# GitHub Copilot Request for Code Review Changes

## Repository Context
- **Repository**: {{.Repository.FullName}}
{{- if .Repository.Description}}
- **Description**: {{.Repository.Description}}
{{- end}}
{{- if .Repository.Language}}
- **Primary Language**: {{.Repository.Language}}
{{- end}}

## Pull Request Context
- **PR #{{.PullRequest.Number}}**: {{.PullRequest.Title}}
- **Author**: {{.PullRequest.Author}}
- **Status**: {{.PullRequest.State}}{{if .PullRequest.IsDraft}} (DRAFT){{end}}{{if .PullRequest.IsMerged}} (MERGED){{end}}
{{- if .PullRequest.Created}}
- **Created**: {{.PullRequest.Created}}
{{- end}}
{{- if .PullRequest.Body}}
- **Description**:
` + "```" + `
{{.PullRequest.Body}}
` + "```" + `
{{- end}}
{{- if .PullRequest.SourceBranch}}
- **Source Branch**: {{.PullRequest.SourceBranch}}
{{- end}}
{{- if .PullRequest.TargetBranch}}
- **Target Branch**: {{.PullRequest.TargetBranch}}
{{- end}}

## Review Comment Context
- **Reviewer**: {{.Comment.Reviewer}}
{{- if .Comment.Date}}
- **Comment Date**: {{.Comment.Date}}
{{- end}}
{{- if .Comment.Path}}
- **File**: ` + "`{{.Comment.Path}}`" + `
{{- if .Comment.LineRange}}
- **Lines**: {{.Comment.LineRange}}
{{- end}}
{{- if .Comment.OriginalLineRange}}
- **Original Lines**: {{.Comment.OriginalLineRange}}
{{- end}}
{{- end}}
{{- if .Comment.DiffHunk}}
- **Code Context**:
` + "```diff" + `
{{.Comment.DiffHunk}}
` + "```" + `
{{- end}}

## Review Comment/Requested Changes
{{- if .Comment.Body}}
` + "```" + `
{{.Comment.Body}}
` + "```" + `
{{- end}}

## Instructions for GitHub Copilot
Based on the above context, please help me address the review comment by:

1. **Understanding the Issue**: Analyze the reviewer's feedback and identify what needs to be changed
2. **Proposing Solutions**: Suggest specific code changes that address the reviewer's concerns
3. **Code Implementation**: Provide the actual code changes needed, with proper formatting and best practices
4. **Explanation**: Explain why the suggested changes address the review feedback
5. **Testing Considerations**: Suggest any additional tests or validation that might be needed

Please focus on:
- Maintaining code quality and consistency with the existing codebase
- Following the project's coding standards and conventions
- Ensuring the changes align with the PR's overall objectives
- Addressing any security, performance, or maintainability concerns raised

## Additional Context
- **Generated**: {{.Generated}}
{{- if .Comment.HTMLURL}}
- **Direct Link**: {{.Comment.HTMLURL}}
{{- end}}`

const simplePromptTemplate = `# Review Comment for {{.Repository.Name}} PR #{{.PullRequest.Number}}

{{- if .Comment.Path}}
**File**: ` + "`{{.Comment.Path}}`" + `{{if .Comment.LineRange}} ({{.Comment.LineRange}}){{end}}

{{- end}}
{{- if .Comment.DiffHunk}}
**Code Context**:
` + "```diff" + `
{{.Comment.DiffHunk}}
` + "```" + `

{{- end}}
**Review Comment**:
{{.Comment.Body}}

**Please help me address this review feedback with specific code changes.**`

// New creates a new prompt generator
func New() *Generator {
	fullTmpl := template.Must(template.New("full").Parse(fullPromptTemplate))
	simpleTmpl := template.Must(template.New("simple").Parse(simplePromptTemplate))

	return &Generator{
		fullTemplate:   fullTmpl,
		simpleTemplate: simpleTmpl,
	}
}

// GenerateFullPrompt creates a full, comprehensive prompt for GitHub Copilot based on PR and comment context
func (g *Generator) GenerateFullPrompt(repo *github.Repository, pr *github.PullRequest, comment *github.PullRequestComment) string {
	data := g.buildTemplateData(repo, pr, comment)

	var buf bytes.Buffer
	if err := g.fullTemplate.Execute(&buf, data); err != nil {
		// Fallback to error message if template execution fails
		return fmt.Sprintf("Error generating prompt: %v", err)
	}

	return buf.String()
}

// GenerateSimplePrompt creates a simple, more focused prompt for GitHub Copilot based on PR and comment context
func (g *Generator) GenerateSimplePrompt(repo *github.Repository, pr *github.PullRequest, comment *github.PullRequestComment) string {
	data := g.buildTemplateData(repo, pr, comment)

	var buf bytes.Buffer
	if err := g.simpleTemplate.Execute(&buf, data); err != nil {
		// Fallback to error message if template execution fails
		return fmt.Sprintf("Error generating prompt: %v", err)
	}

	return buf.String()
}

// buildTemplateData converts GitHub API structs to template-friendly data
func (g *Generator) buildTemplateData(repo *github.Repository, pr *github.PullRequest, comment *github.PullRequestComment) *TemplateData {
	data := &TemplateData{
		Repository: &RepositoryData{
			FullName:    repo.GetFullName(),
			Name:        repo.GetName(),
			Description: repo.GetDescription(),
			Language:    repo.GetLanguage(),
		},
		PullRequest: &PullRequestData{
			Number:   pr.GetNumber(),
			Title:    pr.GetTitle(),
			Author:   pr.GetUser().GetLogin(),
			State:    pr.GetState(),
			IsDraft:  pr.GetDraft(),
			IsMerged: pr.GetMerged(),
			Body:     pr.GetBody(),
		},
		Comment: &CommentData{
			Reviewer:          comment.GetUser().GetLogin(),
			Path:              comment.GetPath(),
			Line:              comment.GetLine(),
			StartLine:         comment.GetStartLine(),
			OriginalLine:      comment.GetOriginalLine(),
			OriginalStartLine: comment.GetOriginalStartLine(),
			DiffHunk:          comment.GetDiffHunk(),
			Body:              comment.GetBody(),
			HTMLURL:           comment.GetHTMLURL(),
		},
		Generated: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Format dates
	if pr.CreatedAt != nil {
		data.PullRequest.Created = pr.CreatedAt.Format("2006-01-02 15:04")
	}
	if comment.CreatedAt != nil {
		data.Comment.Date = comment.CreatedAt.Format("2006-01-02 15:04")
	}

	// Format branch names
	if pr.GetHead() != nil {
		data.PullRequest.SourceBranch = pr.GetHead().GetRef()
	}
	if pr.GetBase() != nil {
		data.PullRequest.TargetBranch = pr.GetBase().GetRef()
	}

	// Format line ranges
	if data.Comment.Line != 0 {
		if data.Comment.StartLine != 0 && data.Comment.StartLine != data.Comment.Line {
			// Multi-line comment
			data.Comment.LineRange = fmt.Sprintf("L%d-%d", data.Comment.StartLine, data.Comment.Line)
		} else {
			// Single line comment
			data.Comment.LineRange = fmt.Sprintf("L%d", data.Comment.Line)
		}
	}

	if data.Comment.OriginalLine != 0 && data.Comment.OriginalLine != data.Comment.Line {
		if data.Comment.OriginalStartLine != 0 && data.Comment.OriginalStartLine != data.Comment.OriginalLine {
			// Multi-line original comment
			data.Comment.OriginalLineRange = fmt.Sprintf("L%d-%d", data.Comment.OriginalStartLine, data.Comment.OriginalLine)
		} else {
			// Single line original comment
			data.Comment.OriginalLineRange = fmt.Sprintf("L%d", data.Comment.OriginalLine)
		}
	}

	return data
}
