package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/fulgidus/terminalpub/internal/auth"
	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/fulgidus/terminalpub/internal/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// OAuthHandler handles OAuth device flow requests
type OAuthHandler struct {
	db                *pgxpool.Pool
	redis             *redis.Client
	cfg               *config.Config
	deviceFlowService *auth.DeviceFlowService
	tokenService      *auth.TokenService
	sshKeyService     *auth.SSHKeyService
	sessionManager    *auth.SessionManager
	userService       *services.UserService
	mastodonService   *auth.MastodonService
	templates         *template.Template
}

// NewOAuthHandler creates a new OAuthHandler instance
func NewOAuthHandler(
	db *pgxpool.Pool,
	redis *redis.Client,
	cfg *config.Config,
) *OAuthHandler {
	// Initialize all services
	mastodonService := auth.NewMastodonService(db, cfg.OAuth.CallbackURL, []string{"read", "write", "follow"})
	deviceFlowService := auth.NewDeviceFlowService(db, fmt.Sprintf("http://%s/device", cfg.Server.Domain))
	tokenService := auth.NewTokenService(db, mastodonService)
	sshKeyService := auth.NewSSHKeyService(db)
	sessionManager := auth.NewSessionManager(db, redis)
	userService := services.NewUserService(db)

	// Load templates
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		log.Printf("Warning: Failed to load templates: %v", err)
		tmpl = template.New("fallback")
	}

	return &OAuthHandler{
		db:                db,
		redis:             redis,
		cfg:               cfg,
		deviceFlowService: deviceFlowService,
		tokenService:      tokenService,
		sshKeyService:     sshKeyService,
		sessionManager:    sessionManager,
		userService:       userService,
		mastodonService:   mastodonService,
		templates:         tmpl,
	}
}

// ServeHTTP handles device authorization requests
func (h *OAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.showDeviceForm(w, r)
		return
	}

	if r.Method == http.MethodPost {
		h.handleDeviceCode(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// showDeviceForm displays the device code entry form
func (h *OAuthHandler) showDeviceForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Error":   "",
		"Success": "",
	}

	if err := h.templates.ExecuteTemplate(w, "device.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleDeviceCode processes the submitted device code
func (h *OAuthHandler) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		h.showError(w, "Invalid form data")
		return
	}

	userCode := strings.TrimSpace(r.FormValue("user_code"))
	if userCode == "" {
		h.showError(w, "User code is required")
		return
	}

	// Normalize code (remove spaces, hyphens, uppercase)
	userCode = strings.ToUpper(strings.ReplaceAll(userCode, "-", ""))

	// Lookup device code
	ctx := r.Context()
	deviceCode, err := h.deviceFlowService.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		h.showError(w, "Invalid or expired code. Please try again from your SSH session.")
		return
	}

	// Check if already authorized
	if deviceCode.Authorized {
		h.showSuccess(w, "This code has already been used. You should be logged in!")
		return
	}

	// Redirect to Mastodon OAuth
	authURL, err := h.tokenService.GetAuthorizationURL(ctx, deviceCode.InstanceURL, userCode)
	if err != nil {
		log.Printf("Failed to generate auth URL: %v", err)
		h.showError(w, "Failed to connect to Mastodon. Please try again.")
		return
	}

	// Redirect user to Mastodon
	http.Redirect(w, r, authURL, http.StatusFound)
}

// showError displays an error message
func (h *OAuthHandler) showError(w http.ResponseWriter, message string) {
	data := map[string]interface{}{
		"Error":   message,
		"Success": "",
	}

	if err := h.templates.ExecuteTemplate(w, "device.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// showSuccess displays a success message
func (h *OAuthHandler) showSuccess(w http.ResponseWriter, message string) {
	data := map[string]interface{}{
		"Error":   "",
		"Success": message,
	}

	if err := h.templates.ExecuteTemplate(w, "device.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleCallback handles the OAuth callback from Mastodon
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state") // state contains the user_code
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		h.showError(w, fmt.Sprintf("Authorization failed: %s", errorParam))
		return
	}

	if code == "" || state == "" {
		h.showError(w, "Missing authorization code or state")
		return
	}

	// Lookup device code using state (which is the user_code)
	deviceCode, err := h.deviceFlowService.GetDeviceCodeByUserCode(ctx, state)
	if err != nil {
		h.showError(w, "Invalid or expired session")
		return
	}

	// Exchange authorization code for access token
	token, err := h.tokenService.ExchangeCodeForToken(ctx, deviceCode.InstanceURL, code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		h.showError(w, "Failed to obtain access token")
		return
	}

	// Create or get existing user
	username := fmt.Sprintf("%s@%s", token.Username, strings.TrimPrefix(deviceCode.InstanceURL, "https://"))
	username = strings.ReplaceAll(username, ".", "_") // Sanitize username

	user, err := h.userService.GetUserByUsername(ctx, username)
	if err != nil {
		// User doesn't exist, create new one
		user, err = h.userService.CreateUser(ctx, username, "")
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			h.showError(w, "Failed to create user account")
			return
		}
	}

	// Store token
	if err := h.tokenService.StoreToken(ctx, user.ID, token, true); err != nil {
		log.Printf("Failed to store token: %v", err)
		h.showError(w, "Failed to store authentication token")
		return
	}

	// Update user's primary Mastodon account
	if err := h.userService.UpdatePrimaryMastodonAccount(ctx, user.ID, deviceCode.InstanceURL, token.MastodonID, token.Username); err != nil {
		log.Printf("Failed to update primary mastodon account: %v", err)
	}

	// Authorize the device code
	if err := h.deviceFlowService.AuthorizeDeviceCode(ctx, state, user.ID); err != nil {
		log.Printf("Failed to authorize device code: %v", err)
		h.showError(w, "Failed to complete authorization")
		return
	}

	// Show success message
	h.showSuccess(w, fmt.Sprintf("✅ Successfully logged in as @%s! You can close this window and return to your SSH session.", token.Username))
}
