package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MastodonService handles Mastodon app registration and OAuth operations
type MastodonService struct {
	db          *pgxpool.Pool
	redirectURI string
	scopes      []string
}

// NewMastodonService creates a new MastodonService instance
func NewMastodonService(db *pgxpool.Pool, redirectURI string, scopes []string) *MastodonService {
	return &MastodonService{
		db:          db,
		redirectURI: redirectURI,
		scopes:      scopes,
	}
}

// MastodonAppResponse represents the response from Mastodon app registration
type MastodonAppResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Website      string `json:"website"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	VapidKey     string `json:"vapid_key"`
}

// MastodonAccountResponse represents a Mastodon account
type MastodonAccountResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Acct        string `json:"acct"`
	DisplayName string `json:"display_name"`
	Note        string `json:"note"`
	Avatar      string `json:"avatar"`
	URL         string `json:"url"`
}

// NormalizeInstanceURL normalizes a Mastodon instance URL
func NormalizeInstanceURL(instance string) string {
	// Remove protocol if present
	instance = strings.TrimPrefix(instance, "https://")
	instance = strings.TrimPrefix(instance, "http://")

	// Remove trailing slashes
	instance = strings.TrimSuffix(instance, "/")

	// Add https:// prefix
	return "https://" + instance
}

// GetOrCreateApp retrieves an existing app registration or creates a new one
func (m *MastodonService) GetOrCreateApp(ctx context.Context, instanceURL string) (*models.MastodonApp, error) {
	instanceURL = NormalizeInstanceURL(instanceURL)

	// Try to get existing app
	app, err := m.getApp(ctx, instanceURL)
	if err == nil {
		return app, nil
	}

	// Register new app
	return m.registerApp(ctx, instanceURL)
}

// getApp retrieves an app registration from the database
func (m *MastodonService) getApp(ctx context.Context, instanceURL string) (*models.MastodonApp, error) {
	query := `
		SELECT id, instance_url, client_id, client_secret, redirect_uri, scopes, created_at, updated_at
		FROM mastodon_apps
		WHERE instance_url = $1
	`

	var app models.MastodonApp
	err := m.db.QueryRow(ctx, query, instanceURL).Scan(
		&app.ID,
		&app.InstanceURL,
		&app.ClientID,
		&app.ClientSecret,
		&app.RedirectURI,
		&app.Scopes,
		&app.CreatedAt,
		&app.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	return &app, nil
}

// registerApp registers a new app with a Mastodon instance
func (m *MastodonService) registerApp(ctx context.Context, instanceURL string) (*models.MastodonApp, error) {
	// Prepare registration request
	payload := map[string]interface{}{
		"client_name":   "terminalpub",
		"redirect_uris": m.redirectURI,
		"scopes":        strings.Join(m.scopes, " "),
		"website":       "https://github.com/fulgidus/terminalpub",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make registration request
	url := fmt.Sprintf("%s/api/v1/apps", instanceURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to register app: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var appResp MastodonAppResponse
	if err := json.NewDecoder(resp.Body).Decode(&appResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Store in database
	query := `
		INSERT INTO mastodon_apps (instance_url, client_id, client_secret, redirect_uri, scopes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	app := &models.MastodonApp{
		InstanceURL:  instanceURL,
		ClientID:     appResp.ClientID,
		ClientSecret: appResp.ClientSecret,
		RedirectURI:  appResp.RedirectURI,
		Scopes:       strings.Join(m.scopes, " "),
	}

	err = m.db.QueryRow(ctx, query,
		app.InstanceURL,
		app.ClientID,
		app.ClientSecret,
		app.RedirectURI,
		app.Scopes,
	).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to store app: %w", err)
	}

	return app, nil
}

// GetAccount retrieves account information from Mastodon
func (m *MastodonService) GetAccount(ctx context.Context, instanceURL, accessToken string) (*MastodonAccountResponse, error) {
	instanceURL = NormalizeInstanceURL(instanceURL)

	url := fmt.Sprintf("%s/api/v1/accounts/verify_credentials", instanceURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get account, status %d: %s", resp.StatusCode, string(body))
	}

	var account MastodonAccountResponse
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode account: %w", err)
	}

	return &account, nil
}
