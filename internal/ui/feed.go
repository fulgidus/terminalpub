package ui

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fulgidus/terminalpub/internal/services"
)

// FeedModel represents the feed view state
type FeedModel struct {
	statuses       []services.MastodonStatus
	selectedIndex  int
	scrollOffset   int
	timelineType   services.TimelineType
	loading        bool
	loadingMore    bool
	err            error
	viewportHeight int
	statusMessage  string
	hasMore        bool
}

// NewFeedModel creates a new feed model
func NewFeedModel() FeedModel {
	return FeedModel{
		statuses:      []services.MastodonStatus{},
		hasMore:       true,
		selectedIndex: 0,
		scrollOffset:  0,
		timelineType:  services.TimelineHome,
		loading:       false,
	}
}

// RenderFeed renders the feed screen
func (m *Model) renderFeed() string {
	if m.feed.loading {
		return m.renderLoadingFeed()
	}

	if m.feed.err != nil {
		return m.renderFeedError()
	}

	if len(m.feed.statuses) == 0 {
		return m.renderEmptyFeed()
	}

	return m.renderFeedWithPosts()
}

// renderLoadingFeed shows a loading spinner
func (m *Model) renderLoadingFeed() string {
	timelineName := getTimelineName(m.feed.timelineType)
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║          %s Timeline                    ║
╠════════════════════════════════════════════╣
║                                            ║
║                Loading...                  ║
║                                            ║
║  Fetching posts from Mastodon...          ║
║                                            ║
╚════════════════════════════════════════════╝
`, timelineName)
}

// renderFeedError shows an error message
func (m *Model) renderFeedError() string {
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║              Feed Error                    ║
╠════════════════════════════════════════════╣
║                                            ║
║  Failed to load timeline:                  ║
║  %s
║                                            ║
║  [R] Retry  [B] Back  [Q] Quit             ║
║                                            ║
╚════════════════════════════════════════════╝
`, m.feed.err.Error())
}

// renderEmptyFeed shows when no posts are available
func (m *Model) renderEmptyFeed() string {
	timelineName := getTimelineName(m.feed.timelineType)
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║          %s Timeline                    ║
╠════════════════════════════════════════════╣
║                                            ║
║  No posts to display                       ║
║                                            ║
║  Try switching to a different timeline:    ║
║  [H] Home  [L] Local  [F] Federated        ║
║                                            ║
║  [B] Back  [Q] Quit                        ║
║                                            ║
╚════════════════════════════════════════════╝
`, timelineName)
}

// renderFeedWithPosts shows the timeline with posts
func (m *Model) renderFeedWithPosts() string {
	var b strings.Builder
	timelineName := getTimelineName(m.feed.timelineType)

	// Use dynamic width (minimum 60, use terminal width - 4 for margins)
	contentWidth := m.width - 4
	if contentWidth < 60 {
		contentWidth = 60
	}
	if contentWidth > 120 {
		contentWidth = 120 // Max width for readability
	}

	// Header
	topLine := "╔" + strings.Repeat("═", contentWidth) + "╗\n"
	titleText := fmt.Sprintf("%s Timeline (%d posts)", timelineName, len(m.feed.statuses))
	titleLine := "║" + centerText(titleText, contentWidth) + "║\n"
	dividerLine := "╠" + strings.Repeat("═", contentWidth) + "╣\n"
	emptyLine := "║" + strings.Repeat(" ", contentWidth) + "║\n"

	b.WriteString(topLine)
	b.WriteString(titleLine)
	b.WriteString(dividerLine)
	b.WriteString(emptyLine)

	// Calculate which posts to show (viewport)
	postsPerPage := (m.height - 12) / 8 // Estimate ~8 lines per post
	if postsPerPage < 3 {
		postsPerPage = 3
	}

	startIdx := m.feed.scrollOffset
	endIdx := startIdx + postsPerPage
	if endIdx > len(m.feed.statuses) {
		endIdx = len(m.feed.statuses)
	}

	// Render visible posts
	for i := startIdx; i < endIdx; i++ {
		status := m.feed.statuses[i]
		isSelected := i == m.feed.selectedIndex

		// Render post with dynamic width
		b.WriteString(m.renderPostDynamic(status, isSelected, contentWidth))
		b.WriteString(emptyLine)
		b.WriteString("║" + strings.Repeat("─", contentWidth) + "║\n")
	}

	// Footer with controls
	statusMsg := m.feed.statusMessage
	if statusMsg == "" {
		if m.feed.loadingMore {
			statusMsg = "Loading more posts..."
		} else if !m.feed.hasMore {
			statusMsg = "No more posts"
		} else {
			statusMsg = "Ready"
		}
	}

	moreHint := ""
	if m.feed.hasMore && !m.feed.loadingMore {
		moreHint = "[M] Load more  "
	}

	b.WriteString(emptyLine)
	controlLine1 := "↑/↓ Navigate  [H]ome [L]ocal [F]ederated"
	controlLine2 := fmt.Sprintf("[X] Like  [S] Boost  [R] Refresh  %s", moreHint)
	controlLine3 := fmt.Sprintf("Post %d/%d  [B]ack  [Q]uit", m.feed.selectedIndex+1, len(m.feed.statuses))
	statusLine := fmt.Sprintf("Status: %s", statusMsg)

	b.WriteString("║ " + padRight(controlLine1, contentWidth-2) + " ║\n")
	b.WriteString("║ " + padRight(controlLine2, contentWidth-2) + " ║\n")
	b.WriteString("║ " + padRight(controlLine3, contentWidth-2) + " ║\n")
	b.WriteString(emptyLine)
	b.WriteString("║ " + padRight(statusLine, contentWidth-2) + " ║\n")
	b.WriteString("╚" + strings.Repeat("═", contentWidth) + "╝\n")

	return b.String()
}

// renderPost renders a single Mastodon post (old fixed-width version)
func (m *Model) renderPost(status services.MastodonStatus, selected bool) string {
	return m.renderPostDynamic(status, selected, 44) // Default 44 for compatibility
}

// renderPostDynamic renders a single Mastodon post with dynamic width
func (m *Model) renderPostDynamic(status services.MastodonStatus, selected bool, width int) string {
	// Handle boost/reblog
	originalStatus := status
	if status.Reblog != nil {
		originalStatus = *status.Reblog
	}

	// Format author
	author := originalStatus.Account.DisplayName
	if author == "" {
		author = originalStatus.Account.Username
	}
	handle := fmt.Sprintf("@%s", originalStatus.Account.Acct)

	// Strip HTML from content
	content := stripHTML(originalStatus.Content)

	// Format metadata
	likes := originalStatus.FavouritesCount
	boosts := originalStatus.ReblogsCount
	replies := originalStatus.RepliesCount

	// Build post display
	var style string
	if selected {
		style = "► " // Selection indicator
	} else {
		style = "  "
	}

	var b strings.Builder
	contentWidth := width - 4 // Account for margins

	// Show if it's a boost
	if status.Reblog != nil {
		boostText := fmt.Sprintf("%s🔄 %s boosted:", style, truncate(status.Account.DisplayName, 20))
		b.WriteString("║ " + padRight(boostText, width-2) + " ║\n")
	}

	// Author and handle
	authorText := fmt.Sprintf("%s%s", style, truncate(author, contentWidth-3))
	b.WriteString("║ " + padRight(authorText, width-2) + " ║\n")

	handleText := fmt.Sprintf("  %s", truncate(handle, contentWidth-2))
	b.WriteString("║ " + padRight(handleText, width-2) + " ║\n")
	b.WriteString("║" + strings.Repeat(" ", width) + "║\n")

	// Content (word-wrapped to dynamic width)
	lines := wrapText(content, contentWidth-2)
	maxContentLines := 5 // Show up to 5 lines of content
	for i, line := range lines {
		if i >= maxContentLines {
			b.WriteString("║ " + padRight("  ...", width-2) + " ║\n")
			break
		}
		b.WriteString("║ " + padRight("  "+line, width-2) + " ║\n")
	}

	b.WriteString("║" + strings.Repeat(" ", width) + "║\n")

	// Interaction stats
	statsText := fmt.Sprintf("  ❤ %-4d  🔄 %-4d  💬 %-4d", likes, boosts, replies)
	b.WriteString("║ " + padRight(statsText, width-2) + " ║\n")

	return b.String()
}

// Helper functions

func getTimelineName(t services.TimelineType) string {
	switch t {
	case services.TimelineHome:
		return "Home"
	case services.TimelineLocal:
		return "Local"
	case services.TimelineFederated:
		return "Federated"
	default:
		return "Unknown"
	}
}

func stripHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile("<[^>]*>")
	s = re.ReplaceAllString(s, "")

	// Decode HTML entities
	s = html.UnescapeString(s)

	// Replace multiple spaces with single space
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func truncateContent(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}

		// Limit to 3 lines max
		if len(lines) >= 3 {
			break
		}
	}

	if currentLine != "" && len(lines) < 3 {
		lines = append(lines, currentLine)
	}

	// Ensure at least one line
	if len(lines) == 0 {
		lines = []string{""}
	}

	return lines
}

// fetchTimelineCmd fetches timeline from Mastodon
func fetchTimelineCmd(ctx *AppContext, userID int, timelineType services.TimelineType, limit int) tea.Cmd {
	return func() tea.Msg {
		mastodonService := services.NewMastodonService(ctx.DB)

		statuses, err := mastodonService.GetTimeline(
			context.Background(),
			userID,
			timelineType,
			limit,
			"", // maxID for pagination
		)

		if err != nil {
			return timelineMsg{err: err}
		}

		return timelineMsg{
			statuses:     statuses,
			timelineType: timelineType,
			isLoadMore:   false,
		}
	}
}

// loadMorePostsCmd loads more posts for pagination
func loadMorePostsCmd(ctx *AppContext, userID int, timelineType services.TimelineType, limit int, maxID string) tea.Cmd {
	return func() tea.Msg {
		mastodonService := services.NewMastodonService(ctx.DB)

		statuses, err := mastodonService.GetTimeline(
			context.Background(),
			userID,
			timelineType,
			limit,
			maxID,
		)

		if err != nil {
			return timelineMsg{err: err, isLoadMore: true}
		}

		return timelineMsg{
			statuses:     statuses,
			timelineType: timelineType,
			isLoadMore:   true,
		}
	}
}

// likeStatusCmd likes a status
func likeStatusCmd(ctx *AppContext, userID int, statusID string) tea.Cmd {
	return func() tea.Msg {
		mastodonService := services.NewMastodonService(ctx.DB)
		err := mastodonService.FavouriteStatus(context.Background(), userID, statusID)
		return likeMsg{err: err}
	}
}

// boostStatusCmd boosts a status
func boostStatusCmd(ctx *AppContext, userID int, statusID string) tea.Cmd {
	return func() tea.Msg {
		mastodonService := services.NewMastodonService(ctx.DB)
		err := mastodonService.BoostStatus(context.Background(), userID, statusID)
		return boostMsg{err: err}
	}
}

// timelineMsg is returned when timeline is fetched
type timelineMsg struct {
	statuses     []services.MastodonStatus
	timelineType services.TimelineType
	isLoadMore   bool
	err          error
}

// likeMsg is returned when a status is liked
type likeMsg struct {
	err error
}

// boostMsg is returned when a status is boosted
type boostMsg struct {
	err error
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	textLen := len(text)
	if textLen >= width {
		return text[:width]
	}
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-textLen-padding)
}

// padRight pads text to the right
func padRight(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	return text + strings.Repeat(" ", width-len(text))
}

// Lipgloss styles
var (
	postStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	selectedPostStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("170")).
				Padding(1, 2)

	authorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	handleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
