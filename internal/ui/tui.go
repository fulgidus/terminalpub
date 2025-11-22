package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/fulgidus/terminalpub/internal/auth"
	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	gossh "golang.org/x/crypto/ssh"
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
)

// Model represents the TUI state
type Model struct {
	ctx           *AppContext
	sshSession    ssh.Session
	screen        screenType
	message       string
	input         string
	deviceAuth    *auth.DeviceAuthResponse
	user          *models.User
	sessionID     string
	publicKey     string
	authenticated bool
	pollingTicker *time.Ticker
}

// NewModel creates a new TUI model
func NewModel(ctx *AppContext, s ssh.Session) Model {
	// Extract SSH public key in authorized_keys format
	publicKey := ""
	if s.PublicKey() != nil {
		publicKey = string(gossh.MarshalAuthorizedKey(s.PublicKey()))
	}

	return Model{
		ctx:        ctx,
		sshSession: s,
		screen:     screenWelcome,
		publicKey:  publicKey,
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

	case authenticatedMsg:
		// User is already authenticated
		m.user = msg.user
		m.authenticated = true
		m.screen = screenAuthenticated
		m.message = fmt.Sprintf("Welcome back, %s!", m.user.Username)
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
		}

	case screenAnonymous:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "b", "B", "esc":
			m.screen = screenWelcome
			m.message = ""
		}
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
			return authenticatedMsg{user: nil}
		}

		// Associate SSH key with user
		if publicKey != "" {
			_, _ = ctx.SSHKeyService.AddSSHKeyToUser(
				context.Background(),
				userID,
				publicKey,
			)
		}

		return authenticatedMsg{user: &user}
	}
}

// View renders the TUI
func (m Model) View() string {
	switch m.screen {
	case screenWelcome:
		return m.renderWelcome()
	case screenLoginInstance:
		return m.renderLoginInstance()
	case screenLoginWaiting:
		return m.renderLoginWaiting()
	case screenAuthenticated:
		return m.renderAuthenticated()
	case screenAnonymous:
		return m.renderAnonymous()
	default:
		// Fallback to welcome screen if unknown state
		m.screen = screenWelcome
		return m.renderWelcome()
	}
}

func (m Model) renderWelcome() string {
	status := "guest"
	if m.authenticated && m.user != nil {
		status = m.user.Username
	}

	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║        Welcome to terminalpub!             ║
║        ActivityPub for terminals           ║
╠════════════════════════════════════════════╣
║                                            ║
║  Connected as: %-27s ║
║                                            ║
║  Press a key to continue:                  ║
║                                            ║
║  [L] Login with Mastodon                   ║
║  [A] Continue anonymously                  ║
║  [Q] Quit                                  ║
║                                            ║
╚════════════════════════════════════════════╝

%s
`, status, m.message)
}

func (m Model) renderLoginInstance() string {
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║        Login with Mastodon                 ║
╠════════════════════════════════════════════╣
║                                            ║
║  Enter your Mastodon instance:            ║
║                                            ║
║  > %-40s ║
║                                            ║
║  Examples:                                 ║
║  • mastodon.social                         ║
║  • mas.to                                  ║
║  • fosstodon.org                           ║
║                                            ║
║  Press [Enter] to continue                 ║
║  Press [Esc] to go back                    ║
║                                            ║
╚════════════════════════════════════════════╝

%s
`, m.input, m.message)
}

func (m Model) renderLoginWaiting() string {
	if m.deviceAuth == nil {
		return "Loading..."
	}

	// Calculate time remaining
	timeRemaining := time.Until(m.deviceAuth.ExpiresAt)
	minutes := int(timeRemaining.Minutes())
	seconds := int(timeRemaining.Seconds()) % 60

	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║        Waiting for Authorization           ║
╠════════════════════════════════════════════╣
║                                            ║
║  1. Open your browser and visit:          ║
║                                            ║
║     http://51.91.97.241/device             ║
║                                            ║
║  2. Enter this code:                       ║
║                                            ║
║     ┌────────────┐                         ║
║     │  %s  │                         ║
║     └────────────┘                         ║
║                                            ║
║  3. Authorize terminalpub access           ║
║                                            ║
║  Waiting for authorization...              ║
║  ⏱  Code expires in: %02d:%02d               ║
║                                            ║
║  [Esc] Cancel                              ║
║                                            ║
╚════════════════════════════════════════════╝

Polling for authorization every 5 seconds...
`, m.deviceAuth.UserCode, minutes, seconds)
}

func (m Model) renderAuthenticated() string {
	username := "Unknown"
	if m.user != nil {
		username = m.user.Username
	}

	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║        🎉 Successfully Logged In!          ║
╠════════════════════════════════════════════╣
║                                            ║
║  Welcome, @%-33s ║
║                                            ║
║  Your SSH key has been associated with     ║
║  your account. Next time you connect,      ║
║  you'll be automatically logged in!        ║
║                                            ║
║  Available features:                       ║
║  • View your Mastodon feed                 ║
║  • Post to the fediverse                   ║
║  • Interact with posts (like, boost)       ║
║  • Chat roulette                           ║
║                                            ║
║  [Coming in Phase 3]                       ║
║                                            ║
║  [Q] Quit                                  ║
║                                            ║
╚════════════════════════════════════════════╝

%s
`, username, m.message)
}

func (m Model) renderAnonymous() string {
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║           Anonymous Mode                   ║
╠════════════════════════════════════════════╣
║                                            ║
║  You're browsing as: anonymous             ║
║                                            ║
║  Available features:                       ║
║  • View public feed                        ║
║  • Chat roulette                           ║
║  • Browse hashtags                         ║
║                                            ║
║  [Coming in Phase 4+]                      ║
║                                            ║
║  Commands:                                 ║
║  [B] Back to menu                          ║
║  [Q] Quit                                  ║
║                                            ║
╚════════════════════════════════════════════╝

%s

🚧 This is a work in progress!
Phase 1: Infrastructure ✅
Phase 2: Authentication ✅ (In Progress)
Phase 3: ActivityPub Integration
`, m.message)
}
