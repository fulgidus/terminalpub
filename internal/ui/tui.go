package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/fulgidus/terminalpub/internal/auth"
	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/fulgidus/terminalpub/internal/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	gossh "golang.org/x/crypto/ssh"
)

// Color styles for the UI
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))  // Cyan
	keyStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208")) // Orange
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))             // Green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))              // Red
	subtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))            // Gray
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))             // Bright Blue
)

// AppContext holds shared services for the TUI
type AppContext struct {
	DB                *pgxpool.Pool
	Redis             *redis.Client
	Config            *config.Config
	DeviceFlowService *auth.DeviceFlowService
	SSHKeyService     *auth.SSHKeyService
	SessionManager    *auth.SessionManager
}

// screenType represents different screens in the TUI
type screenType int

const (
	screenWelcome screenType = iota
	screenLogin
	screenLoginInstance
	screenLoginWaiting
	screenAuthenticated
	screenAnonymous
	screenFeed
	screenCompose
	screenThread
)

// Model represents the TUI state
type Model struct {
	ctx            *AppContext
	sshSession     ssh.Session
	screen         screenType
	message        string
	input          string
	deviceAuth     *auth.DeviceAuthResponse
	user           *models.User
	sessionID      string
	publicKey      string
	authenticated  bool
	pollingTicker  *time.Ticker
	feed           FeedModel
	compose        ComposeModel
	thread         ThreadModel
	mastodonSvc    *services.MastodonService
	width          int
	height         int
	returnToScreen screenType // Screen to return to after composing
}

// NewModel creates a new TUI model
func NewModel(ctx *AppContext, s ssh.Session) Model {
	// Extract SSH public key in authorized_keys format
	publicKey := ""
	if s.PublicKey() != nil {
		publicKey = string(gossh.MarshalAuthorizedKey(s.PublicKey()))
	} else {
	}

	return Model{
		ctx:            ctx,
		sshSession:     s,
		screen:         screenWelcome,
		publicKey:      publicKey,
		feed:           NewFeedModel(),
		compose:        NewComposeModel(),
		mastodonSvc:    services.NewMastodonService(ctx.DB),
		width:          80, // Default width
		height:         24, // Default height
		returnToScreen: screenAuthenticated,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Check if user is already authenticated via SSH key
	if m.publicKey != "" && m.ctx.SSHKeyService != nil {
		return checkSSHKeyCmd(m.ctx, m.publicKey)
	}
	return nil
}

// checkSSHKeyCmd checks if SSH key is associated with a user
func checkSSHKeyCmd(ctx *AppContext, publicKey string) tea.Cmd {
	return func() tea.Msg {
		user, err := ctx.SSHKeyService.GetUserBySSHKey(context.Background(), publicKey)
		if err == nil {
			return authenticatedMsg{user: user}
		}
		return nil
	}
}

// Messages
type authenticatedMsg struct {
	user *models.User
}

type deviceCodeMsg struct {
	auth *auth.DeviceAuthResponse
	err  error
}

type pollResultMsg struct {
	authorized bool
	userID     int
	err        error
}

type tickMsg time.Time

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// Update window dimensions
		m.width = msg.Width
		m.height = msg.Height
		m.feed.viewportHeight = msg.Height - 10 // Reserve space for header/footer
		return m, nil

	case authenticatedMsg:
		// User is already authenticated
		m.user = msg.user
		m.authenticated = true
		m.screen = screenAuthenticated
		return m, nil

	case deviceCodeMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v\n\nPress [Esc] to go back", msg.err)
			m.screen = screenLoginInstance
			return m, nil
		}
		m.deviceAuth = msg.auth
		m.screen = screenLoginWaiting
		// Start polling
		return m, tickCmd()

	case pollResultMsg:
		if msg.err != nil {
			// Continue polling
			return m, tickCmd()
		}
		if msg.authorized {
			// User authorized! Load user info
			return m, loadUserCmd(m.ctx, msg.userID, m.publicKey, m.deviceAuth.DeviceCode)
		}
		// Continue polling
		return m, tickCmd()

	case tickMsg:
		// Poll for authorization
		if m.screen == screenLoginWaiting && m.deviceAuth != nil {
			return m, pollAuthorizationCmd(m.ctx, m.deviceAuth.DeviceCode)
		}
		return m, nil

	case timelineMsg:
		// Timeline fetched
		m.feed.loading = false
		m.feed.loadingMore = false

		if msg.err != nil {
			m.feed.err = msg.err
			m.feed.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			if msg.isLoadMore {
				// Append new posts to existing ones
				m.feed.statuses = append(m.feed.statuses, msg.statuses...)
				m.feed.statusMessage = fmt.Sprintf("Loaded %d more posts", len(msg.statuses))

				// Check if we got fewer posts than requested (no more available)
				if len(msg.statuses) < 20 {
					m.feed.hasMore = false
					m.feed.statusMessage = "All posts loaded"
				}
			} else {
				// Replace with new timeline
				m.feed.statuses = msg.statuses
				m.feed.timelineType = msg.timelineType
				m.feed.selectedIndex = 0
				m.feed.scrollOffset = 0
				m.feed.err = nil
				m.feed.hasMore = len(msg.statuses) >= 20
				m.feed.statusMessage = "Timeline loaded"
			}
		}
		return m, nil

	case likeMsg:
		// Status liked/favourited
		if msg.err != nil {
			m.feed.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.feed.statusMessage = "Post liked!"
		}
		return m, nil

	case boostMsg:
		// Status boosted/reblogged
		if msg.err != nil {
			m.feed.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.feed.statusMessage = "Post boosted!"
		}
		return m, nil

	case postStatusMsg:
		// Handle post status request from compose screen
		return m, executePostStatusCmd(m.ctx, m.mastodonSvc, m.user.ID, msg.content, string(msg.visibility), msg.replyToID, msg.contentWarning)

	case postStatusResultMsg:
		// Post completed (success or error) - update compose model
		m.compose.posting = false
		if msg.err != nil {
			m.compose.status = fmt.Sprintf("Error: %v", msg.err)
			m.compose.err = msg.err
		} else {
			// Success - return to previous screen
			m.screen = m.returnToScreen
			m.message = "Post created successfully!"
			// Refresh feed if we're returning to feed
			if m.returnToScreen == screenFeed {
				m.feed.loading = true
				return m, fetchTimelineCmd(m.ctx, m.user.ID, m.feed.timelineType, 20)
			}
		}
		return m, nil

	case composeCancelMsg:
		// User cancelled compose - return to previous screen
		m.screen = m.returnToScreen
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenWelcome:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "l", "L":
			// Check if database is available before allowing login
			if m.ctx == nil || m.ctx.DeviceFlowService == nil {
				m.message = "Login unavailable: Database not connected"
				return m, nil
			}
			m.screen = screenLoginInstance
			m.input = ""
			m.message = ""
		case "a", "A":
			m.screen = screenAnonymous
			m.message = "Anonymous mode activated!"
		}

	case screenLoginInstance:
		switch msg.String() {
		case "enter":
			if m.input != "" {
				// Check if AppContext is available
				if m.ctx == nil || m.ctx.DeviceFlowService == nil {
					m.message = "Error: Database connection not available\n\nPress [Esc] to go back"
					return m, nil
				}
				instance := strings.TrimSpace(m.input)
				m.message = "Connecting to Mastodon..."
				return m, initiateDeviceFlowCmd(m.ctx, instance, m.sshSession.User())
			}
		case "esc", "ctrl+c":
			m.screen = screenWelcome
			m.input = ""
			m.message = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			// Add character to input
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}

	case screenLoginWaiting:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			m.screen = screenWelcome
			m.deviceAuth = nil
		}

	case screenAuthenticated:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "x", "X":
			// Logout - reset to welcome screen
			m.authenticated = false
			m.user = nil
			m.screen = screenWelcome
			m.message = "Logged out successfully"
			return m, nil
		case "f", "F":
			// Open feed screen
			m.screen = screenFeed
			m.feed.loading = true
			m.feed.err = nil
			m.feed.timelineType = services.TimelineHome
			return m, fetchTimelineCmd(m.ctx, m.user.ID, services.TimelineHome, 20)
		case "p", "P":
			// Open compose screen for new post
			m.compose = NewComposeModel()
			m.compose.width = m.width
			m.compose.height = m.height
			m.returnToScreen = screenAuthenticated
			m.screen = screenCompose
			return m, m.compose.Init()
		}

	case screenAnonymous:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "b", "B", "esc":
			m.screen = screenWelcome
			m.message = ""
		}

	case screenFeed:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "b", "B", "esc":
			m.screen = screenAuthenticated
			return m, nil
		case "up", "k":
			// Navigate up
			if m.feed.selectedIndex > 0 {
				m.feed.selectedIndex--
				// Adjust scroll offset if needed
				if m.feed.selectedIndex < m.feed.scrollOffset {
					m.feed.scrollOffset = m.feed.selectedIndex
				}
			}
		case "down", "j":
			// Navigate down
			if m.feed.selectedIndex < len(m.feed.statuses)-1 {
				m.feed.selectedIndex++
				// Adjust scroll offset if needed (viewport shows 5 posts)
				if m.feed.selectedIndex >= m.feed.scrollOffset+5 {
					m.feed.scrollOffset = m.feed.selectedIndex - 4
				}

				// Infinite scrolling: auto-load more when near the end
				postsRemaining := len(m.feed.statuses) - m.feed.selectedIndex
				if postsRemaining <= 5 && m.feed.hasMore && !m.feed.loadingMore && !m.feed.loading {
					// Trigger auto-load
					lastPost := m.feed.statuses[len(m.feed.statuses)-1]
					maxID := lastPost.ID
					m.feed.loadingMore = true
					m.feed.statusMessage = "Loading more..."
					return m, loadMorePostsCmd(m.ctx, m.user.ID, m.feed.timelineType, 20, maxID)
				}
			}
		case "h", "H":
			// Switch to Home timeline
			m.feed.loading = true
			m.feed.timelineType = services.TimelineHome
			return m, fetchTimelineCmd(m.ctx, m.user.ID, services.TimelineHome, 20)
		case "l", "L":
			// Switch to Local timeline
			m.feed.loading = true
			m.feed.timelineType = services.TimelineLocal
			return m, fetchTimelineCmd(m.ctx, m.user.ID, services.TimelineLocal, 20)
		case "f", "F":
			// Switch to Federated timeline
			m.feed.loading = true
			m.feed.timelineType = services.TimelineFederated
			return m, fetchTimelineCmd(m.ctx, m.user.ID, services.TimelineFederated, 20)
		case "ctrl+r":
			// Refresh feed
			m.feed.loading = true
			m.feed.statusMessage = "Refreshing..."
			return m, fetchTimelineCmd(m.ctx, m.user.ID, m.feed.timelineType, 20)

		case "x", "X":
			// Like the selected post (x for love)
			if m.feed.selectedIndex < len(m.feed.statuses) {
				status := m.feed.statuses[m.feed.selectedIndex]
				// If it's a reblog, like the original post
				if status.Reblog != nil {
					return m, likeStatusCmd(m.ctx, m.user.ID, status.Reblog.ID)
				}
				return m, likeStatusCmd(m.ctx, m.user.ID, status.ID)
			}
		case "s", "S":
			// Boost the selected post (s for share)
			if m.feed.selectedIndex < len(m.feed.statuses) {
				status := m.feed.statuses[m.feed.selectedIndex]
				// If it's a reblog, boost the original post
				if status.Reblog != nil {
					return m, boostStatusCmd(m.ctx, m.user.ID, status.Reblog.ID)
				}
				return m, boostStatusCmd(m.ctx, m.user.ID, status.ID)
			}
		case "r", "R":
			// Reply to selected post
			if m.feed.selectedIndex < len(m.feed.statuses) {
				status := m.feed.statuses[m.feed.selectedIndex]
				// If it's a reblog, reply to the original post
				originalStatus := &status
				if status.Reblog != nil {
					originalStatus = status.Reblog
				}
				// Create reply compose model
				author := originalStatus.Account.Acct
				// Strip HTML from content for context display
				content := stripHTML(originalStatus.Content)
				m.compose = NewReplyModel(originalStatus.ID, author, content)
				m.compose.width = m.width
				m.compose.height = m.height
				m.returnToScreen = screenFeed
				m.screen = screenCompose
				return m, m.compose.Init()
			}
		case "t", "T":
			// View thread for selected post
			if m.feed.selectedIndex < len(m.feed.statuses) {
				status := m.feed.statuses[m.feed.selectedIndex]
				// If it's a reblog, view the thread of the original post
				originalStatus := &status
				if status.Reblog != nil {
					originalStatus = status.Reblog
				}
				// Create thread model with background context
				bgCtx := context.Background()
				m.thread = NewThreadModel(bgCtx, m.user.ID, m.mastodonSvc, *originalStatus)
				m.thread.width = m.width
				m.thread.height = m.height
				m.returnToScreen = screenFeed
				m.screen = screenThread
				return m, m.thread.Init()
			}
		}

	case screenCompose:
		// Delegate all compose screen updates to compose model
		var cmd tea.Cmd
		m.compose, cmd = m.compose.Update(msg)
		return m, cmd

	case screenThread:
		// Handle thread screen keys
		switch msg.String() {
		case "esc":
			// Return to feed
			m.screen = m.returnToScreen
			return m, nil
		case "up", "k":
			// Navigate up in thread
			if m.thread.selectedIndex > 0 {
				m.thread.selectedIndex--
			}
		case "down", "j":
			// Navigate down in thread
			if m.thread.selectedIndex < len(m.thread.flattenedThread)-1 {
				m.thread.selectedIndex++
			}
		case "r", "R":
			// Reply to selected post in thread
			if selectedStatus := m.thread.GetSelectedStatus(); selectedStatus != nil {
				author := selectedStatus.Account.Acct
				content := stripHTML(selectedStatus.Content)
				m.compose = NewReplyModel(selectedStatus.ID, author, content)
				m.compose.width = m.width
				m.compose.height = m.height
				m.returnToScreen = screenThread
				m.screen = screenCompose
				return m, m.compose.Init()
			}
		case "o", "O":
			// Open in browser (placeholder for now)
			if selectedStatus := m.thread.GetSelectedStatus(); selectedStatus != nil && selectedStatus.URL != "" {
				m.thread.statusMessage = fmt.Sprintf("URL: %s", selectedStatus.URL)
			}
		}
		// Delegate other updates to thread model
		var cmd tea.Cmd
		m.thread, cmd = m.thread.Update(msg)
		return m, cmd
	}

	return m, nil
}

// initiateDeviceFlowCmd starts the OAuth Device Flow
func initiateDeviceFlowCmd(ctx *AppContext, instance, sessionID string) tea.Cmd {
	return func() tea.Msg {
		// Add timeout to prevent hanging
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		auth, err := ctx.DeviceFlowService.InitiateDeviceFlow(
			bgCtx,
			instance,
			sessionID,
		)
		if err != nil {
			// Wrap error with more context
			return deviceCodeMsg{
				auth: nil,
				err:  fmt.Errorf("failed to connect to %s: %w", instance, err),
			}
		}
		return deviceCodeMsg{auth: auth, err: nil}
	}
}

// pollAuthorizationCmd polls for device authorization
func pollAuthorizationCmd(ctx *AppContext, deviceCode string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(5 * time.Second) // Poll every 5 seconds

		authorized, userID, err := ctx.DeviceFlowService.PollDeviceCode(
			context.Background(),
			deviceCode,
		)

		if err != nil {
			return pollResultMsg{err: err}
		}

		return pollResultMsg{
			authorized: authorized,
			userID:     userID,
		}
	}
}

// tickCmd creates a tick message
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadUserCmd loads user info and associates SSH key
func loadUserCmd(ctx *AppContext, userID int, publicKey, deviceCode string) tea.Cmd {
	return func() tea.Msg {

		// Get user
		var user models.User
		err := ctx.DB.QueryRow(
			context.Background(),
			`SELECT id, username, email, primary_mastodon_instance,
			        primary_mastodon_acct, created_at
			 FROM users WHERE id = $1`,
			userID,
		).Scan(&user.ID, &user.Username, &user.Email, &user.PrimaryMastodonInstance,
			&user.PrimaryMastodonAcct, &user.CreatedAt)

		if err != nil {
			fmt.Printf("Failed to load user: %v\n", err)
			return authenticatedMsg{user: nil}
		}

		// Associate SSH key with user
		if publicKey != "" {
			key, err := ctx.SSHKeyService.AddSSHKeyToUser(
				context.Background(),
				userID,
				publicKey,
			)
			if err != nil {
				fmt.Printf("Failed to save SSH key: %v\n", err)
			} else {
				fmt.Printf("SSH key saved successfully: ID=%d, fingerprint=%s\n", key.ID, key.Fingerprint)
			}
		} else {
		}

		return authenticatedMsg{user: &user}
	}
}

// executePostStatusCmd posts a status to Mastodon
func executePostStatusCmd(ctx *AppContext, mastodonSvc *services.MastodonService, userID int, content, visibility, replyToID, contentWarning string) tea.Cmd {
	return func() tea.Msg {
		statusID, err := mastodonSvc.PostStatus(
			context.Background(),
			userID,
			content,
			visibility,
			replyToID,
			contentWarning,
		)
		return postStatusResultMsg{
			statusID: statusID,
			err:      err,
		}
	}
}

// View renders the TUI
func (m Model) View() string {
	var content string
	switch m.screen {
	case screenWelcome:
		content = m.renderWelcome()
	case screenLoginInstance:
		content = m.renderLoginInstance()
	case screenLoginWaiting:
		content = m.renderLoginWaiting()
	case screenAuthenticated:
		content = m.renderAuthenticated()
	case screenAnonymous:
		content = m.renderAnonymous()
	case screenFeed:
		return m.renderFeed() // Feed uses full screen
	case screenCompose:
		return m.centerContent(m.compose.View())
	case screenThread:
		return m.thread.View()
	default:
		// Fallback to welcome screen if unknown state
		m.screen = screenWelcome
		content = m.renderWelcome()
	}

	// Center content for non-feed screens
	return m.centerContent(content)
}

// centerContent centers content both horizontally and vertically
func (m Model) centerContent(content string) string {
	lines := strings.Split(content, "\n")

	// Calculate content dimensions
	contentHeight := len(lines)
	contentWidth := 0
	for _, line := range lines {
		// Strip ANSI codes for width calculation
		cleanLine := lipgloss.NewStyle().Render(line)
		if len(cleanLine) > contentWidth {
			contentWidth = len(cleanLine)
		}
	}

	// Calculate vertical padding
	verticalPadding := (m.height - contentHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	// Calculate horizontal padding
	horizontalPadding := (m.width - contentWidth) / 2
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	var b strings.Builder

	// Add top padding
	for i := 0; i < verticalPadding; i++ {
		b.WriteString("\n")
	}

	// Add content with horizontal padding
	for _, line := range lines {
		b.WriteString(strings.Repeat(" ", horizontalPadding))
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Add bottom padding to fill screen
	remainingLines := m.height - verticalPadding - contentHeight
	for i := 0; i < remainingLines; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderWelcome() string {
	status := "guest"
	if m.authenticated && m.user != nil {
		status = m.user.Username
	}

	var b strings.Builder

	width := 60 // Fixed content width

	// Title
	title := titleStyle.Render("terminalpub")
	subtitle := subtleStyle.Render("ActivityPub for terminals")
	b.WriteString(centerText(title, width) + "\n")
	b.WriteString(centerText(subtitle, width) + "\n\n")

	// Status
	statusLine := fmt.Sprintf("Connected as: %s", subtleStyle.Render(status))
	b.WriteString(centerText(statusLine, width) + "\n\n")

	// Options
	b.WriteString(centerText(keyStyle.Render("[L]")+" Login with Mastodon", width) + "\n")
	b.WriteString(centerText(keyStyle.Render("[A]")+" Continue anonymously", width) + "\n")
	b.WriteString(centerText(keyStyle.Render("[Q]")+" Quit", width) + "\n")

	if m.message != "" {
		b.WriteString("\n")
		msgStyle := subtleStyle
		if strings.Contains(m.message, "success") {
			msgStyle = successStyle
		} else if strings.Contains(m.message, "Error") {
			msgStyle = errorStyle
		}
		b.WriteString(centerText(msgStyle.Render(m.message), width) + "\n")
	}

	return b.String()
}

func (m Model) renderLoginInstance() string {
	var b strings.Builder
	width := 60

	// Title
	b.WriteString(centerText(titleStyle.Render("Login with Mastodon"), width) + "\n\n")

	// Prompt
	b.WriteString(centerText("Enter your Mastodon instance:", width) + "\n")
	b.WriteString(centerText(promptStyle.Render("> "+m.input+"█"), width) + "\n\n")

	// Examples
	b.WriteString(centerText(subtleStyle.Render("Examples: mastodon.social, mas.to, fosstodon.org"), width) + "\n\n")

	// Instructions
	b.WriteString(centerText(keyStyle.Render("[Enter]")+" to continue  "+keyStyle.Render("[Esc]")+" to go back", width) + "\n")

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(centerText(errorStyle.Render(m.message), width) + "\n")
	}

	return b.String()
}

func (m Model) renderLoginWaiting() string {
	if m.deviceAuth == nil {
		return "Loading..."
	}

	// Calculate time remaining
	timeRemaining := time.Until(m.deviceAuth.ExpiresAt)
	minutes := int(timeRemaining.Minutes())
	seconds := int(timeRemaining.Seconds()) % 60

	var b strings.Builder
	width := 60

	// Title
	b.WriteString(centerText(titleStyle.Render("Waiting for Authorization"), width) + "\n\n")

	// Instructions
	b.WriteString(centerText("1. Open your browser and visit:", width) + "\n")
	b.WriteString(centerText(promptStyle.Bold(true).Render("http://51.91.97.241/device"), width) + "\n\n")

	b.WriteString(centerText("2. Enter this code:", width) + "\n")
	b.WriteString(centerText(promptStyle.Bold(true).Render(m.deviceAuth.UserCode), width) + "\n\n")

	b.WriteString(centerText("3. Authorize terminalpub access", width) + "\n\n")

	// Status
	b.WriteString(centerText(subtleStyle.Render("Waiting for authorization..."), width) + "\n")
	expiryText := fmt.Sprintf("Code expires in: %02d:%02d", minutes, seconds)
	b.WriteString(centerText(subtleStyle.Render(expiryText), width) + "\n\n")

	b.WriteString(centerText(keyStyle.Render("[Esc]")+" Cancel", width) + "\n")

	return b.String()
}

func (m Model) renderAuthenticated() string {
	username := "Unknown"
	if m.user != nil {
		username = m.user.Username
	}

	var b strings.Builder
	width := 60

	// Welcome message
	welcomeMsg := fmt.Sprintf("Welcome, %s", titleStyle.Render("@"+username))
	b.WriteString(centerText(welcomeMsg, width) + "\n\n")

	b.WriteString(centerText(subtleStyle.Render("Your SSH key has been associated with your account."), width) + "\n")
	b.WriteString(centerText(subtleStyle.Render("Next time you connect, you'll be automatically logged in!"), width) + "\n\n")

	// Menu options
	b.WriteString(centerText(keyStyle.Render("[P]")+" Compose new post", width) + "\n")
	b.WriteString(centerText(keyStyle.Render("[F]")+" View your Mastodon feed", width) + "\n")
	b.WriteString(centerText(keyStyle.Render("[X]")+" Logout", width) + "\n")
	b.WriteString(centerText(keyStyle.Render("[Q]")+" Quit", width) + "\n")

	if m.message != "" {
		b.WriteString("\n")
		msgStyle := successStyle
		if strings.Contains(m.message, "Error") {
			msgStyle = errorStyle
		}
		b.WriteString(centerText(msgStyle.Render(m.message), width) + "\n")
	}

	// Bottom line
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	return b.String()
}

func (m Model) renderAnonymous() string {
	var b strings.Builder

	b.WriteString(strings.Repeat("─", m.width) + "\n\n")
	b.WriteString("  Anonymous Mode\n\n")
	b.WriteString("  You're browsing as: anonymous\n\n")
	b.WriteString("  Available features:\n")
	b.WriteString("  • View public feed\n")
	b.WriteString("  • Browse hashtags\n")
	b.WriteString("  [Coming soon...]\n\n")
	b.WriteString("  [B] Back to menu  [Q] Quit\n\n")

	if m.message != "" {
		b.WriteString("  " + m.message + "\n\n")
	}

	b.WriteString(strings.Repeat("─", m.width) + "\n")

	return b.String()
}
