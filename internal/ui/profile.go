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

// ProfileModel represents the user profile view state
type ProfileModel struct {
	ctx             context.Context
	userID          int
	mastodonService *services.MastodonService
	accountID       string
	account         *services.MastodonAccount
	statuses        []services.MastodonStatus
	relationship    *services.AccountRelationship
	selectedIndex   int
	scrollOffset    int
	loading         bool
	statusMessage   string
	width           int
	height          int
	err             error
}

// profileLoadedMsg is sent when profile data is fetched
type profileLoadedMsg struct {
	account      *services.MastodonAccount
	statuses     []services.MastodonStatus
	relationship *services.AccountRelationship
	err          error
}

// followActionMsg is sent when follow/unfollow action completes
type followActionMsg struct {
	following bool
	err       error
}

// NewProfileModel creates a new profile view model
func NewProfileModel(ctx context.Context, userID int, mastodonService *services.MastodonService, accountID string) ProfileModel {
	return ProfileModel{
		ctx:             ctx,
		userID:          userID,
		mastodonService: mastodonService,
		accountID:       accountID,
		loading:         true,
		statusMessage:   "Loading profile...",
	}
}

// Init initializes the profile model and fetches profile data
func (m ProfileModel) Init() tea.Cmd {
	return m.fetchProfileCmd()
}

// Update handles messages for the profile view
func (m ProfileModel) Update(msg tea.Msg) (ProfileModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case profileLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}

		m.account = msg.account
		m.statuses = msg.statuses
		m.relationship = msg.relationship
		m.statusMessage = ""
		return m, nil

	case followActionMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			if msg.following {
				m.statusMessage = "Followed user!"
				if m.relationship != nil {
					m.relationship.Following = true
				}
			} else {
				m.statusMessage = "Unfollowed user"
				if m.relationship != nil {
					m.relationship.Following = false
				}
			}
		}
		return m, nil
	}

	return m, nil
}

// View renders the profile view
func (m ProfileModel) View() string {
	if m.loading {
		return m.statusMessage
	}

	if m.err != nil {
		return fmt.Sprintf("Error loading profile: %v\n\nPress ESC to go back", m.err)
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	b.WriteString(titleStyle.Render("User Profile") + "\n\n")

	// User info
	cyanColor := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	grayColor := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	greenColor := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// Display name
	displayName := m.account.DisplayName
	if displayName == "" {
		displayName = m.account.Username
	}
	b.WriteString(cyanColor.Render(displayName) + "\n")
	b.WriteString(grayColor.Render("@"+m.account.Acct) + "\n\n")

	// Bio (strip HTML)
	if m.account.Note != "" {
		bio := stripHTMLProfile(m.account.Note)
		if len(bio) > 200 {
			bio = bio[:197] + "..."
		}
		b.WriteString(bio + "\n\n")
	}

	// Stats
	stats := fmt.Sprintf("Following: %d   Followers: %d   Posts: %d",
		m.account.FollowingCount,
		m.account.FollowersCount,
		m.account.StatusesCount)
	b.WriteString(stats + "\n\n")

	// Follow button
	if m.relationship != nil {
		if m.relationship.Following {
			b.WriteString(greenColor.Render("[Following ✓]") + "\n\n")
		} else {
			b.WriteString(grayColor.Render("[Not Following]") + "\n\n")
		}
	}

	// Recent posts section
	b.WriteString(grayColor.Render(strings.Repeat("─", 40)) + "\n")
	b.WriteString(titleStyle.Render("Recent Posts") + "\n")
	b.WriteString(grayColor.Render(strings.Repeat("─", 40)) + "\n\n")

	// Calculate available height for posts
	headerLines := 15 // approximate header size
	footerLines := 2
	availableHeight := m.height - headerLines - footerLines
	if availableHeight < 3 {
		availableHeight = 3
	}

	// Render posts
	postsPerScreen := availableHeight / 3 // Each post takes ~3 lines
	if postsPerScreen < 1 {
		postsPerScreen = 1
	}

	// Ensure scroll offset keeps selected item visible
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}
	if m.selectedIndex >= m.scrollOffset+postsPerScreen {
		m.scrollOffset = m.selectedIndex - postsPerScreen + 1
	}

	endIndex := m.scrollOffset + postsPerScreen
	if endIndex > len(m.statuses) {
		endIndex = len(m.statuses)
	}

	selectionColor := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	for i := m.scrollOffset; i < endIndex; i++ {
		status := m.statuses[i]
		selector := "  "
		if i == m.selectedIndex {
			selector = selectionColor.Render("► ")
		}

		// Content
		content := stripHTMLProfile(status.Content)
		if len(content) > 150 {
			content = content[:147] + "..."
		}
		b.WriteString(selector + content + "\n")

		// Stats
		stats := fmt.Sprintf("Likes: %d  Boosts: %d  Replies: %d",
			status.FavouritesCount,
			status.ReblogsCount,
			status.RepliesCount)
		b.WriteString(selector + grayColor.Render(stats) + "\n")

		if i < endIndex-1 {
			b.WriteString(selector + grayColor.Render("────────────────────────────") + "\n")
		}
	}

	// Controls
	b.WriteString("\n")
	keyColor := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	subtleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	followText := "Follow"
	if m.relationship != nil && m.relationship.Following {
		followText = "Unfollow"
	}

	controls := fmt.Sprintf("  %s Navigate  %s %s  %s Reply  %s Thread  %s Back",
		subtleColor.Render("↑/↓"),
		keyColor.Render("[F]"),
		followText,
		keyColor.Render("[R]"),
		keyColor.Render("[T]"),
		keyColor.Render("[ESC]"))
	b.WriteString(controls)

	if m.statusMessage != "" {
		statusColor := greenColor
		if strings.Contains(m.statusMessage, "Error") {
			statusColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		}
		b.WriteString("\n  " + statusColor.Render(m.statusMessage))
	}

	return b.String()
}

// fetchProfileCmd fetches profile data
func (m ProfileModel) fetchProfileCmd() tea.Cmd {
	return func() tea.Msg {
		// Fetch account info
		account, err := m.mastodonService.GetAccount(m.ctx, m.userID, m.accountID)
		if err != nil {
			return profileLoadedMsg{err: err}
		}

		// Fetch recent statuses
		statuses, err := m.mastodonService.GetAccountStatuses(m.ctx, m.userID, m.accountID, 20)
		if err != nil {
			return profileLoadedMsg{err: err}
		}

		// Fetch relationship
		relationship, err := m.mastodonService.GetAccountRelationship(m.ctx, m.userID, m.accountID)
		if err != nil {
			// Relationship fetch is not critical, continue without it
			relationship = nil
		}

		return profileLoadedMsg{
			account:      account,
			statuses:     statuses,
			relationship: relationship,
		}
	}
}

// GetSelectedStatus returns the currently selected status
func (m ProfileModel) GetSelectedStatus() *services.MastodonStatus {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.statuses) {
		return &m.statuses[m.selectedIndex]
	}
	return nil
}

// stripHTMLProfile removes HTML tags from content (profile-specific version)
func stripHTMLProfile(content string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	stripped := re.ReplaceAllString(content, "")

	// Decode HTML entities
	stripped = html.UnescapeString(stripped)

	// Replace multiple spaces/newlines with single space
	stripped = strings.Join(strings.Fields(stripped), " ")

	return stripped
}
