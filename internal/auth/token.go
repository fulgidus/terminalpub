package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TokenService handles OAuth token operations
type TokenService struct {
	db              *pgxpool.Pool
	mastodonService *MastodonService
}

// NewTokenService creates a new TokenService instance
func NewTokenService(db *pgxpool.Pool, mastodonService *MastodonService) *TokenService {
	return &TokenService{
		db:              db,
		mastodonService: mastodonService,
	}
}

// MastodonTokenResponse represents the OAuth token response from Mastodon
type MastodonTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func (t *TokenService) ExchangeCodeForToken(ctx context.Context, instanceURL, code string) (*models.MastodonToken, error) {
	instanceURL = NormalizeInstanceURL(instanceURL)

	// Get app credentials
	app, err := t.mastodonService.GetOrCreateApp(ctx, instanceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Prepare token request
	data := url.Values{
		"client_id":     {app.ClientID},
		"client_secret": {app.ClientSecret},
		"redirect_uri":  {app.RedirectURI},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"scope":         {app.Scopes},
	}

	// Make token request
	tokenURL := fmt.Sprintf("%s/oauth/token", instanceURL)
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResp MastodonTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Get account information
	account, err := t.mastodonService.GetAccount(ctx, instanceURL, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	// Calculate expiration
	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &expiry
	}

	token := &models.MastodonToken{
		InstanceURL:  instanceURL,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Scopes:       tokenResp.Scope,
		ExpiresAt:    expiresAt,
		MastodonID:   account.ID,
		Username:     account.Username,
		DisplayName:  account.DisplayName,
		AvatarURL:    account.Avatar,
	}

	return token, nil
}

// StoreToken stores or updates a Mastodon token for a user
func (t *TokenService) StoreToken(ctx context.Context, userID int, token *models.MastodonToken, isPrimary bool) error {
	// If this is marked as primary, unset other primary tokens
	if isPrimary {
		_, err := t.db.Exec(ctx,
			"UPDATE mastodon_tokens SET is_primary = FALSE WHERE user_id = $1",
			userID,
		)
		if err != nil {
			return fmt.Errorf("failed to unset primary tokens: %w", err)
		}
	}

	// Insert or update token
	query := `
		INSERT INTO mastodon_tokens (
			user_id, instance_url, access_token, refresh_token, token_type,
			scopes, expires_at, mastodon_id, username, display_name, avatar_url, is_primary
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, instance_url, mastodon_id)
		DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_type = EXCLUDED.token_type,
			scopes = EXCLUDED.scopes,
			expires_at = EXCLUDED.expires_at,
			username = EXCLUDED.username,
			display_name = EXCLUDED.display_name,
			avatar_url = EXCLUDED.avatar_url,
			is_primary = EXCLUDED.is_primary,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`

	err := t.db.QueryRow(ctx, query,
		userID,
		token.InstanceURL,
		token.AccessToken,
		token.RefreshToken,
		token.TokenType,
		token.Scopes,
		token.ExpiresAt,
		token.MastodonID,
		token.Username,
		token.DisplayName,
		token.AvatarURL,
		isPrimary,
	).Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	token.UserID = userID
	token.IsPrimary = isPrimary

	return nil
}

// GetPrimaryToken retrieves the primary Mastodon token for a user
func (t *TokenService) GetPrimaryToken(ctx context.Context, userID int) (*models.MastodonToken, error) {
	query := `
		SELECT id, user_id, instance_url, access_token, refresh_token, token_type,
		       scopes, expires_at, mastodon_id, username, display_name, avatar_url,
		       is_primary, created_at, updated_at
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = TRUE
	`

	var token models.MastodonToken
	err := t.db.QueryRow(ctx, query, userID).Scan(
		&token.ID,
		&token.UserID,
		&token.InstanceURL,
		&token.AccessToken,
		&token.RefreshToken,
		&token.TokenType,
		&token.Scopes,
		&token.ExpiresAt,
		&token.MastodonID,
		&token.Username,
		&token.DisplayName,
		&token.AvatarURL,
		&token.IsPrimary,
		&token.CreatedAt,
		&token.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("primary token not found: %w", err)
	}

	return &token, nil
}

// RefreshToken refreshes an expired Mastodon token
func (t *TokenService) RefreshToken(ctx context.Context, token *models.MastodonToken) (*models.MastodonToken, error) {
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// Get app credentials
	app, err := t.mastodonService.GetOrCreateApp(ctx, token.InstanceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Prepare refresh request
	data := url.Values{
		"client_id":     {app.ClientID},
		"client_secret": {app.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"scope":         {token.Scopes},
	}

	// Make refresh request
	tokenURL := fmt.Sprintf("%s/oauth/token", token.InstanceURL)
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResp MastodonTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Update token
	token.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		token.RefreshToken = tokenResp.RefreshToken
	}

	if tokenResp.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		token.ExpiresAt = &expiry
	}

	// Store updated token
	if err := t.StoreToken(ctx, token.UserID, token, token.IsPrimary); err != nil {
		return nil, fmt.Errorf("failed to store refreshed token: %w", err)
	}

	return token, nil
}

// GetAuthorizationURL generates the Mastodon OAuth authorization URL
func (t *TokenService) GetAuthorizationURL(ctx context.Context, instanceURL, state string) (string, error) {
	instanceURL = NormalizeInstanceURL(instanceURL)

	// Get or create app
	app, err := t.mastodonService.GetOrCreateApp(ctx, instanceURL)
	if err != nil {
		return "", fmt.Errorf("failed to get app: %w", err)
	}

	// Build authorization URL
	params := url.Values{
		"client_id":     {app.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {app.RedirectURI},
		"scope":         {app.Scopes},
		"state":         {state},
	}

	authURL := fmt.Sprintf("%s/oauth/authorize?%s", instanceURL, params.Encode())
	return authURL, nil
}
