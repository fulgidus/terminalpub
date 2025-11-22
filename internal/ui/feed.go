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
	err            error
	viewportHeight int
	statusMessage  string
}

// NewFeedModel creates a new feed model
func NewFeedModel() FeedModel {
	return FeedModel{
		statuses:      []services.MastodonStatus{},
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

	// Header
	b.WriteString(fmt.Sprintf(`╔════════════════════════════════════════════╗
║       %s Timeline (%d posts)         ║
╠════════════════════════════════════════════╣
║                                            ║
`, timelineName, len(m.feed.statuses)))

	// Calculate which posts to show (viewport)
	startIdx := m.feed.scrollOffset
	endIdx := startIdx + 5 // Show 5 posts at a time
	if endIdx > len(m.feed.statuses) {
		endIdx = len(m.feed.statuses)
	}

	// Render visible posts
	for i := startIdx; i < endIdx; i++ {
		status := m.feed.statuses[i]
		isSelected := i == m.feed.selectedIndex

		// Render post
		b.WriteString(m.renderPost(status, isSelected))
		b.WriteString("║                                            ║\n")
		b.WriteString("║────────────────────────────────────────────║\n")
	}

	// Footer with controls
	statusMsg := m.feed.statusMessage
	if statusMsg == "" {
		statusMsg = "Ready"
	}
	b.WriteString(fmt.Sprintf(`║                                            ║
║  ↑/↓ Navigate  [H]ome [L]ocal [F]ederated ║
║  [X] Like  [S] Boost  [R] Refresh          ║
║  Post %d/%d  [B]ack  [Q]uit               ║
║                                            ║
║  Status: %-34s ║
╚════════════════════════════════════════════╝
`, m.feed.selectedIndex+1, len(m.feed.statuses), truncate(statusMsg, 34)))

	return b.String()
}

// renderPost renders a single Mastodon post
func (m *Model) renderPost(status services.MastodonStatus, selected bool) string {
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

	// Truncate content to fit in terminal (max 40 chars per line, 3 lines)
	content = truncateContent(content, 120)

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

	// Show if it's a boost
	if status.Reblog != nil {
		b.WriteString(fmt.Sprintf("║ %s🔄 %s boosted:                      ║\n", style, status.Account.DisplayName))
	}

	b.WriteString(fmt.Sprintf("║ %s%-30s ║\n", style, truncate(author, 28)))
	b.WriteString(fmt.Sprintf("║   %s                              ║\n", truncate(handle, 40)))
	b.WriteString("║                                            ║\n")

	// Content (word-wrapped)
	lines := wrapText(content, 40)
	for _, line := range lines {
		b.WriteString(fmt.Sprintf("║   %-40s ║\n", line))
	}

	b.WriteString("║                                            ║\n")
	b.WriteString(fmt.Sprintf("║   ❤ %-4d  🔄 %-4d  💬 %-4d              ║\n", likes, boosts, replies))

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
