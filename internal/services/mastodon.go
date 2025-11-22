package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TimelineType represents different types of Mastodon timelines
type TimelineType string

const (
	// TimelineHome - Home timeline (posts from accounts you follow)
	TimelineHome TimelineType = "home"
	// TimelineLocal - Local timeline (public posts from your instance)
	TimelineLocal TimelineType = "local"
	// TimelineFederated - Federated/Global timeline (public posts from all known instances)
	TimelineFederated TimelineType = "public"
)

// MastodonService handles communication with Mastodon APIs
type MastodonService struct {
	db     *pgxpool.Pool
	client *http.Client
}

// NewMastodonService creates a new MastodonService instance
func NewMastodonService(db *pgxpool.Pool) *MastodonService {
	return &MastodonService{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MastodonStatus represents a Mastodon post/status
type MastodonStatus struct {
	ID                 string            `json:"id"`
	CreatedAt          time.Time         `json:"created_at"`
	Content            string            `json:"content"`
	Visibility         string            `json:"visibility"`
	Sensitive          bool              `json:"sensitive"`
	SpoilerText        string            `json:"spoiler_text"`
	ReblogsCount       int               `json:"reblogs_count"`
	FavouritesCount    int               `json:"favourites_count"`
	RepliesCount       int               `json:"replies_count"`
	URL                string            `json:"url"`
	InReplyToID        *string           `json:"in_reply_to_id"`
	InReplyToAccountID *string           `json:"in_reply_to_account_id"`
	Reblog             *MastodonStatus   `json:"reblog"`
	Account            MastodonAccount   `json:"account"`
	MediaAttachments   []MastodonMedia   `json:"media_attachments"`
	Mentions           []MastodonMention `json:"mentions"`
	Tags               []MastodonTag     `json:"tags"`
	Card               *MastodonCard     `json:"card"`
	Favourited         bool              `json:"favourited"`
	Reblogged          bool              `json:"reblogged"`
	Bookmarked         bool              `json:"bookmarked"`
}

// MastodonAccount represents a Mastodon account
type MastodonAccount struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Acct           string    `json:"acct"`
	DisplayName    string    `json:"display_name"`
	Note           string    `json:"note"`
	URL            string    `json:"url"`
	Avatar         string    `json:"avatar"`
	Header         string    `json:"header"`
	FollowersCount int       `json:"followers_count"`
	FollowingCount int       `json:"following_count"`
	StatusesCount  int       `json:"statuses_count"`
	CreatedAt      time.Time `json:"created_at"`
	Bot            bool      `json:"bot"`
	Locked         bool      `json:"locked"`
}

// MastodonMedia represents a media attachment
type MastodonMedia struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	PreviewURL  string `json:"preview_url"`
	Description string `json:"description"`
}

// MastodonMention represents a mention in a status
type MastodonMention struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Acct     string `json:"acct"`
	URL      string `json:"url"`
}

// MastodonTag represents a hashtag
type MastodonTag struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// MastodonCard represents a link preview card
type MastodonCard struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Image       string `json:"image"`
}

// GetHomeTimeline fetches the home timeline for a user (convenience method)
func (s *MastodonService) GetHomeTimeline(ctx context.Context, userID int, limit int, maxID string) ([]MastodonStatus, error) {
	return s.GetTimeline(ctx, userID, TimelineHome, limit, maxID)
}

// GetTimeline fetches any timeline type (home, local, or federated)
func (s *MastodonService) GetTimeline(ctx context.Context, userID int, timelineType TimelineType, limit int, maxID string) ([]MastodonStatus, error) {
	// Get the user's primary Mastodon token
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	return s.fetchTimeline(ctx, instanceURL, accessToken, timelineType, limit, maxID)
}

// GetPublicTimeline fetches the public/federated timeline (for anonymous users)
func (s *MastodonService) GetPublicTimeline(ctx context.Context, instanceURL string, local bool, limit int, maxID string) ([]MastodonStatus, error) {
	timelineType := TimelineFederated
	if local {
		timelineType = TimelineLocal
	}
	return s.fetchTimeline(ctx, instanceURL, "", timelineType, limit, maxID)
}

// fetchTimeline is a helper function to fetch any timeline
func (s *MastodonService) fetchTimeline(ctx context.Context, instanceURL, accessToken string, timelineType TimelineType, limit int, maxID string) ([]MastodonStatus, error) {
	// Build API URL based on timeline type
	var apiURL string
	switch timelineType {
	case TimelineHome:
		apiURL = fmt.Sprintf("%s/api/v1/timelines/home?limit=%d", instanceURL, limit)
	case TimelineLocal:
		apiURL = fmt.Sprintf("%s/api/v1/timelines/public?local=true&limit=%d", instanceURL, limit)
	case TimelineFederated:
		apiURL = fmt.Sprintf("%s/api/v1/timelines/public?limit=%d", instanceURL, limit)
	default:
		return nil, fmt.Errorf("invalid timeline type: %s", timelineType)
	}

	if maxID != "" {
		apiURL += fmt.Sprintf("&max_id=%s", maxID)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header if provided (for authenticated requests)
	if accessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var statuses []MastodonStatus
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return statuses, nil
}

// FavouriteStatus likes/favourites a status
func (s *MastodonService) FavouriteStatus(ctx context.Context, userID int, statusID string) error {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/statuses/%s/favourite", instanceURL, statusID)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to favourite status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// BoostStatus reblogs/boosts a status
func (s *MastodonService) BoostStatus(ctx context.Context, userID int, statusID string) error {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/statuses/%s/reblog", instanceURL, statusID)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to boost status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PostStatusRequest represents the request body for posting a status
type PostStatusRequest struct {
	Status      string `json:"status"`
	Visibility  string `json:"visibility,omitempty"`
	InReplyToID string `json:"in_reply_to_id,omitempty"`
	SpoilerText string `json:"spoiler_text,omitempty"`
}

// PostStatus creates a new status (post) on Mastodon
func (s *MastodonService) PostStatus(ctx context.Context, userID int, content, visibility, inReplyToID, contentWarning string) (string, error) {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return "", fmt.Errorf("failed to get user token: %w", err)
	}

	// Build request body
	reqBody := PostStatusRequest{
		Status:      content,
		Visibility:  visibility,
		InReplyToID: inReplyToID,
		SpoilerText: contentWarning,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	apiURL := fmt.Sprintf("%s/api/v1/statuses", instanceURL)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to post status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get status ID
	var status MastodonStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return status.ID, nil
}

// StatusContext represents the context of a status (ancestors and descendants)
type StatusContext struct {
	Ancestors   []MastodonStatus `json:"ancestors"`
	Descendants []MastodonStatus `json:"descendants"`
}

// GetStatusContext fetches the context (thread) for a given status
func (s *MastodonService) GetStatusContext(ctx context.Context, userID int, statusID string) (*StatusContext, error) {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/statuses/%s/context", instanceURL, statusID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch status context: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	var context StatusContext
	if err := json.NewDecoder(resp.Body).Decode(&context); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &context, nil
}

// GetAccount fetches account information for a given account ID
func (s *MastodonService) GetAccount(ctx context.Context, userID int, accountID string) (*MastodonAccount, error) {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/accounts/%s", instanceURL, accountID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	var account MastodonAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &account, nil
}

// GetAccountStatuses fetches recent statuses for a given account
func (s *MastodonService) GetAccountStatuses(ctx context.Context, userID int, accountID string, limit int) ([]MastodonStatus, error) {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/accounts/%s/statuses?limit=%d", instanceURL, accountID, limit)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account statuses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	var statuses []MastodonStatus
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return statuses, nil
}

// AccountRelationship represents the relationship between the current user and another account
type AccountRelationship struct {
	ID         string `json:"id"`
	Following  bool   `json:"following"`
	FollowedBy bool   `json:"followed_by"`
	Blocking   bool   `json:"blocking"`
	Muting     bool   `json:"muting"`
	Requested  bool   `json:"requested"`
}

// GetAccountRelationship fetches the relationship with a given account
func (s *MastodonService) GetAccountRelationship(ctx context.Context, userID int, accountID string) (*AccountRelationship, error) {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/accounts/relationships?id[]=%s", instanceURL, accountID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch relationship: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	var relationships []AccountRelationship
	if err := json.NewDecoder(resp.Body).Decode(&relationships); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(relationships) == 0 {
		return nil, fmt.Errorf("no relationship found")
	}

	return &relationships[0], nil
}

// FollowAccount follows a given account
func (s *MastodonService) FollowAccount(ctx context.Context, userID int, accountID string) error {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/accounts/%s/follow", instanceURL, accountID)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to follow account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UnfollowAccount unfollows a given account
func (s *MastodonService) UnfollowAccount(ctx context.Context, userID int, accountID string) error {
	var accessToken, instanceURL string
	err := s.db.QueryRow(ctx, `
		SELECT access_token, instance_url
		FROM mastodon_tokens
		WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(&accessToken, &instanceURL)

	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/accounts/%s/unfollow", instanceURL, accountID)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to unfollow account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mastodon API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
