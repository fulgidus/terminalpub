package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ComposeMode indicates whether user is composing a new post or replying
type ComposeMode int

const (
	ComposeNew ComposeMode = iota
	ComposeReply
)

// VisibilityOption represents Mastodon post visibility settings
type VisibilityOption string

const (
	VisibilityPublic   VisibilityOption = "public"
	VisibilityUnlisted VisibilityOption = "unlisted"
	VisibilityPrivate  VisibilityOption = "private" // followers-only
	VisibilityDirect   VisibilityOption = "direct"
)

// ComposeModel represents the compose screen state
type ComposeModel struct {
	textarea       textarea.Model
	mode           ComposeMode
	replyToID      string
	replyToAuthor  string
	replyToContent string
	visibility     VisibilityOption
	contentWarning string
	cwEnabled      bool
	width          int
	height         int
	status         string
	posting        bool
	posted         bool
	err            error
}

// NewComposeModel creates a new compose screen model
func NewComposeModel() ComposeModel {
	ta := textarea.New()
	ta.Placeholder = "What's on your mind?"
	ta.Focus()
	ta.CharLimit = 500 // Mastodon default character limit
	ta.ShowLineNumbers = false

	return ComposeModel{
		textarea:   ta,
		mode:       ComposeNew,
		visibility: VisibilityPublic,
		width:      80,
		height:     24,
	}
}

// NewReplyModel creates a compose model for replying to a post
func NewReplyModel(replyToID, replyToAuthor, replyToContent string) ComposeModel {
	m := NewComposeModel()
	m.mode = ComposeReply
	m.replyToID = replyToID
	m.replyToAuthor = replyToAuthor
	m.replyToContent = replyToContent

	// Pre-populate with @mention
	if replyToAuthor != "" {
		m.textarea.SetValue("@" + replyToAuthor + " ")
	}

	return m
}

// Init initializes the compose model
func (m ComposeModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the compose screen
func (m ComposeModel) Update(msg tea.Msg) (ComposeModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel and return to previous screen
			return m, func() tea.Msg {
				return composeCancelMsg{}
			}

		case "ctrl+p":
			// Post the status
			if m.posting {
				return m, nil // Already posting
			}
			content := m.textarea.Value()
			if strings.TrimSpace(content) == "" {
				m.status = "Cannot post empty status"
				return m, nil
			}
			if len(content) > 500 {
				m.status = "Status exceeds 500 characters"
				return m, nil
			}
			m.posting = true
			m.status = "Posting..."
			return m, postStatusCmd(content, m.visibility, m.replyToID, m.contentWarning)

		case "ctrl+w":
			// Toggle content warning
			m.cwEnabled = !m.cwEnabled
			if !m.cwEnabled {
				m.contentWarning = ""
			}
			return m, nil

		case "ctrl+v":
			// Cycle visibility
			m.visibility = m.nextVisibility()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(m.width - 6)    // Account for padding and borders
		m.textarea.SetHeight(m.height - 15) // Account for header, footer, and controls

	case postStatusResultMsg:
		m.posting = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
			m.err = msg.err
		} else {
			m.posted = true
			m.status = "Posted successfully!"
			// Return to previous screen after a brief delay
			return m, func() tea.Msg {
				return composeSuccessMsg{statusID: msg.statusID}
			}
		}
		return m, nil
	}

	// Update textarea
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the compose screen
func (m ComposeModel) View() string {
	var b strings.Builder

	// Determine title based on mode
	title := "Compose New Post"
	if m.mode == ComposeReply {
		title = "Reply to Post"
	}

	// Use dynamic width constraints
	minWidth := 60
	maxWidth := 100
	contentWidth := m.width
	if contentWidth < minWidth {
		contentWidth = minWidth
	}
	if contentWidth > maxWidth {
		contentWidth = maxWidth
	}

	// Top border with title
	topBorder := "╔" + strings.Repeat("═", contentWidth-2) + "╗"
	titleLine := "║" + centerText(title, contentWidth-2) + "║"
	separator := "╠" + strings.Repeat("═", contentWidth-2) + "╣"
	bottomBorder := "╚" + strings.Repeat("═", contentWidth-2) + "╝"

	b.WriteString(topBorder + "\n")
	b.WriteString(titleLine + "\n")
	b.WriteString(separator + "\n")

	// If replying, show context
	if m.mode == ComposeReply {
		b.WriteString("║" + strings.Repeat(" ", contentWidth-2) + "║\n")
		b.WriteString("║  Replying to:" + strings.Repeat(" ", contentWidth-16) + "║\n")
		b.WriteString("║  " + padRight("┌"+strings.Repeat("─", contentWidth-6)+"┐", contentWidth-2) + "║\n")

		// Show reply context (truncated)
		authorLine := fmt.Sprintf("│ %s", m.replyToAuthor)
		b.WriteString("║  " + padRight(authorLine, contentWidth-4) + "  ║\n")

		// Truncate content if too long
		contentLines := wrapText(m.replyToContent, contentWidth-10)
		maxContextLines := 3
		for i, line := range contentLines {
			if i >= maxContextLines {
				b.WriteString("║  " + padRight("│ ...", contentWidth-4) + "  ║\n")
				break
			}
			b.WriteString("║  " + padRight(fmt.Sprintf("│ %s", line), contentWidth-4) + "  ║\n")
		}

		b.WriteString("║  " + padRight("└"+strings.Repeat("─", contentWidth-6)+"┘", contentWidth-2) + "║\n")
		b.WriteString("║" + strings.Repeat(" ", contentWidth-2) + "║\n")
	}

	// Textarea section
	b.WriteString("║" + strings.Repeat(" ", contentWidth-2) + "║\n")
	if m.mode == ComposeReply {
		b.WriteString("║  Your reply:" + strings.Repeat(" ", contentWidth-15) + "║\n")
	} else {
		b.WriteString("║  Write your post:" + strings.Repeat(" ", contentWidth-20) + "║\n")
	}

	// Render textarea with border
	textareaLines := strings.Split(m.textarea.View(), "\n")
	b.WriteString("║  " + padRight("┌"+strings.Repeat("─", contentWidth-6)+"┐", contentWidth-2) + "║\n")
	for _, line := range textareaLines {
		// Ensure line fits within box
		if len(line) > contentWidth-8 {
			line = line[:contentWidth-8]
		}
		b.WriteString("║  " + padRight("│ "+line, contentWidth-4) + "  ║\n")
	}
	b.WriteString("║  " + padRight("└"+strings.Repeat("─", contentWidth-6)+"┘", contentWidth-2) + "║\n")

	b.WriteString("║" + strings.Repeat(" ", contentWidth-2) + "║\n")

	// Character count
	charCount := len(m.textarea.Value())
	charLimit := m.textarea.CharLimit
	charStyle := lipgloss.NewStyle()
	if charCount > charLimit {
		charStyle = charStyle.Foreground(lipgloss.Color("9")) // Red
	}
	charCountStr := charStyle.Render(fmt.Sprintf("Characters: %d/%d", charCount, charLimit))
	b.WriteString("║  " + padRight(charCountStr, contentWidth-2) + "║\n")

	b.WriteString("║" + strings.Repeat(" ", contentWidth-2) + "║\n")

	// Visibility selector
	visibilityStr := fmt.Sprintf("Visibility: [%s ▼]", m.visibility)
	b.WriteString("║  " + padRight(visibilityStr, contentWidth-2) + "║\n")

	// Content warning
	cwStr := "Content Warning: [ ] Add CW"
	if m.cwEnabled {
		cwStr = "Content Warning: [✓] CW Enabled"
	}
	b.WriteString("║  " + padRight(cwStr, contentWidth-2) + "║\n")

	b.WriteString("║" + strings.Repeat(" ", contentWidth-2) + "║\n")

	// Keyboard shortcuts
	shortcuts := "[Ctrl+P] Post  [Ctrl+W] Toggle CW  [Ctrl+V] Visibility  [Esc] Cancel"
	b.WriteString("║  " + padRight(shortcuts, contentWidth-2) + "║\n")

	b.WriteString("║" + strings.Repeat(" ", contentWidth-2) + "║\n")

	// Status message
	if m.status != "" {
		statusStyle := lipgloss.NewStyle()
		if strings.Contains(m.status, "Error") {
			statusStyle = statusStyle.Foreground(lipgloss.Color("9")) // Red
		} else if strings.Contains(m.status, "success") {
			statusStyle = statusStyle.Foreground(lipgloss.Color("10")) // Green
		}
		statusStr := statusStyle.Render("Status: " + m.status)
		b.WriteString("║  " + padRight(statusStr, contentWidth-2) + "║\n")
	} else {
		b.WriteString("║  " + padRight("Status: Ready", contentWidth-2) + "║\n")
	}

	b.WriteString(bottomBorder)

	return b.String()
}

// nextVisibility cycles to the next visibility option
func (m ComposeModel) nextVisibility() VisibilityOption {
	switch m.visibility {
	case VisibilityPublic:
		return VisibilityUnlisted
	case VisibilityUnlisted:
		return VisibilityPrivate
	case VisibilityPrivate:
		return VisibilityDirect
	case VisibilityDirect:
		return VisibilityPublic
	default:
		return VisibilityPublic
	}
}

// Messages for compose screen
type composeCancelMsg struct{}

type composeSuccessMsg struct {
	statusID string
}

type postStatusResultMsg struct {
	statusID string
	err      error
}

// postStatusCmd posts a status to Mastodon
func postStatusCmd(content string, visibility VisibilityOption, replyToID string, contentWarning string) tea.Cmd {
	return func() tea.Msg {
		// This will be implemented in tui.go to access the app context
		// For now, return a placeholder
		return postStatusMsg{
			content:        content,
			visibility:     visibility,
			replyToID:      replyToID,
			contentWarning: contentWarning,
		}
	}
}

type postStatusMsg struct {
	content        string
	visibility     VisibilityOption
	replyToID      string
	contentWarning string
}
