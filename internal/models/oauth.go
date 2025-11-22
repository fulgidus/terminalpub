package models

import "time"

// MastodonApp represents a registered Mastodon application
// One app registration per Mastodon instance
type MastodonApp struct {
	ID           int       `json:"id"`
	InstanceURL  string    `json:"instance_url"`  // e.g., https://mastodon.social
	ClientID     string    `json:"client_id"`     // App client ID from Mastodon
	ClientSecret string    `json:"client_secret"` // App client secret from Mastodon
	RedirectURI  string    `json:"redirect_uri"`  // OAuth redirect URI
	Scopes       string    `json:"scopes"`        // Space-separated OAuth scopes
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DeviceCode represents a pending device authorization flow
// Generated when a user starts the OAuth Device Flow
type DeviceCode struct {
	ID              int       `json:"id"`
	UserCode        string    `json:"user_code"`        // 8-character code shown to user (e.g., "WXYZ-1234")
	DeviceCode      string    `json:"device_code"`      // Long device code for polling
	InstanceURL     string    `json:"instance_url"`     // Mastodon instance URL
	SSHSessionID    string    `json:"ssh_session_id"`   // SSH session waiting for auth
	VerificationURI string    `json:"verification_uri"` // URI where user enters code
	ExpiresAt       time.Time `json:"expires_at"`       // Code expiration time (typically 15 minutes)
	Authorized      bool      `json:"authorized"`       // Whether user has authorized
	UserID          *int      `json:"user_id"`          // Set after authorization completes
	CreatedAt       time.Time `json:"created_at"`
}

// MastodonToken stores OAuth tokens for a user's Mastodon account
type MastodonToken struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`       // terminalpub user ID
	InstanceURL  string     `json:"instance_url"`  // Mastodon instance
	AccessToken  string     `json:"access_token"`  // OAuth access token
	RefreshToken string     `json:"refresh_token"` // OAuth refresh token (if supported)
	TokenType    string     `json:"token_type"`    // Usually "Bearer"
	Scopes       string     `json:"scopes"`        // Granted scopes
	ExpiresAt    *time.Time `json:"expires_at"`    // Token expiration (if provided)
	MastodonID   string     `json:"mastodon_id"`   // User's Mastodon account ID
	Username     string     `json:"username"`      // Mastodon username
	DisplayName  string     `json:"display_name"`  // Mastodon display name
	AvatarURL    string     `json:"avatar_url"`    // Mastodon avatar
	IsPrimary    bool       `json:"is_primary"`    // Is this the primary account?
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Session represents an active SSH session
type Session struct {
	ID         string    `json:"id"`         // Session ID (UUID)
	UserID     *int      `json:"user_id"`    // User ID (null for anonymous)
	PublicKey  string    `json:"public_key"` // SSH public key
	IPAddress  string    `json:"ip_address"` // Client IP
	Anonymous  bool      `json:"anonymous"`  // Is anonymous session?
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"` // Last activity timestamp
	ExpiresAt  time.Time `json:"expires_at"`   // Session expiration
}
