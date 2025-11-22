package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivityPubHandler handles ActivityPub-related HTTP requests
type ActivityPubHandler struct {
	db     *pgxpool.Pool
	config *config.Config
}

// NewActivityPubHandler creates a new ActivityPub handler
func NewActivityPubHandler(db *pgxpool.Pool, cfg *config.Config) *ActivityPubHandler {
	return &ActivityPubHandler{
		db:     db,
		config: cfg,
	}
}

// WebFinger handles WebFinger requests (/.well-known/webfinger)
func (h *ActivityPubHandler) WebFinger(w http.ResponseWriter, r *http.Request) {
	// Get resource parameter
	resource := r.URL.Query().Get("resource")
	if resource == "" {
		http.Error(w, "Missing resource parameter", http.StatusBadRequest)
		return
	}

	// Parse resource (should be acct:username@domain)
	if !strings.HasPrefix(resource, "acct:") {
		http.Error(w, "Invalid resource format", http.StatusBadRequest)
		return
	}

	acct := strings.TrimPrefix(resource, "acct:")
	parts := strings.Split(acct, "@")
	if len(parts) != 2 {
		http.Error(w, "Invalid account format", http.StatusBadRequest)
		return
	}

	username := parts[0]
	domain := parts[1]

	// Verify domain matches our server
	if domain != h.config.Server.Domain {
		http.Error(w, "Unknown domain", http.StatusNotFound)
		return
	}

	// Look up user in database
	ctx := r.Context()
	var user models.User
	err := h.db.QueryRow(ctx,
		"SELECT id, username, bio, created_at FROM users WHERE username = $1",
		username,
	).Scan(&user.ID, &user.Username, &user.Bio, &user.CreatedAt)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Build WebFinger response
	response := map[string]any{
		"subject": resource,
		"aliases": []string{
			fmt.Sprintf("%s/users/%s", h.config.Server.BaseURL, username),
		},
		"links": []map[string]any{
			{
				"rel":  "self",
				"type": "application/activity+json",
				"href": fmt.Sprintf("%s/users/%s", h.config.Server.BaseURL, username),
			},
			{
				"rel":  "http://webfinger.net/rel/profile-page",
				"type": "text/html",
				"href": fmt.Sprintf("%s/@%s", h.config.Server.BaseURL, username),
			},
		},
	}

	w.Header().Set("Content-Type", "application/jrd+json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

// Actor handles Actor endpoint requests (/users/{username})
func (h *ActivityPubHandler) Actor(w http.ResponseWriter, r *http.Request) {
	// Extract username from URL path
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	username := strings.Split(path, "/")[0]

	if username == "" {
		http.Error(w, "Missing username", http.StatusBadRequest)
		return
	}

	// Look up user in database
	ctx := r.Context()
	var user models.User
	err := h.db.QueryRow(ctx,
		"SELECT id, username, bio, private_key, public_key, created_at FROM users WHERE username = $1",
		username,
	).Scan(&user.ID, &user.Username, &user.Bio, &user.PrivateKey, &user.PublicKey, &user.CreatedAt)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Build Actor object
	actorID := fmt.Sprintf("%s/users/%s", h.config.Server.BaseURL, username)

	actor := models.Actor{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		ID:                        actorID,
		Type:                      "Person",
		PreferredUsername:         user.Username,
		Name:                      user.Username,
		Summary:                   user.Bio,
		Inbox:                     fmt.Sprintf("%s/inbox", actorID),
		Outbox:                    fmt.Sprintf("%s/outbox", actorID),
		Followers:                 fmt.Sprintf("%s/followers", actorID),
		Following:                 fmt.Sprintf("%s/following", actorID),
		URL:                       fmt.Sprintf("%s/@%s", h.config.Server.BaseURL, username),
		ManuallyApprovesFollowers: false,
		Published:                 user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		PublicKey: models.ActorPublicKey{
			ID:           fmt.Sprintf("%s#main-key", actorID),
			Owner:        actorID,
			PublicKeyPem: user.PublicKey,
		},
		Endpoints: map[string]any{
			"sharedInbox": fmt.Sprintf("%s/inbox", h.config.Server.BaseURL),
		},
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(actor)
}

// Inbox handles incoming ActivityPub activities (/users/{username}/inbox)
func (h *ActivityPubHandler) Inbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract username from URL path
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "inbox" {
		http.Error(w, "Invalid inbox path", http.StatusBadRequest)
		return
	}
	username := parts[0]

	// Look up user
	ctx := r.Context()
	var userID int
	err := h.db.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// TODO: Verify HTTP signature

	// Parse activity
	var activity map[string]any
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Store activity in database for processing
	activityJSON, _ := json.Marshal(activity)
	activityType, _ := activity["type"].(string)
	actorID, _ := activity["actor"].(string)

	var objectID string
	if obj, ok := activity["object"].(string); ok {
		objectID = obj
	} else if obj, ok := activity["object"].(map[string]any); ok {
		if id, ok := obj["id"].(string); ok {
			objectID = id
		}
	}

	_, err = h.db.Exec(ctx, `
		INSERT INTO activities (user_id, activity_type, actor_id, object_id, activity_json, direction, processed)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, activityType, actorID, objectID, activityJSON, "inbound", false)

	if err != nil {
		http.Error(w, "Failed to store activity", http.StatusInternalServerError)
		return
	}

	// Return 202 Accepted
	w.WriteHeader(http.StatusAccepted)
}

// Outbox handles outbox requests (/users/{username}/outbox)
func (h *ActivityPubHandler) Outbox(w http.ResponseWriter, r *http.Request) {
	// Extract username from URL path
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "outbox" {
		http.Error(w, "Invalid outbox path", http.StatusBadRequest)
		return
	}
	username := parts[0]

	// Look up user
	ctx := r.Context()
	var userID int
	err := h.db.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get page parameter
	page := r.URL.Query().Get("page")

	actorID := fmt.Sprintf("%s/users/%s", h.config.Server.BaseURL, username)
	outboxURL := fmt.Sprintf("%s/outbox", actorID)

	if page == "" {
		// Return OrderedCollection
		var totalItems int
		h.db.QueryRow(ctx, "SELECT COUNT(*) FROM posts WHERE user_id = $1", userID).Scan(&totalItems)

		collection := models.OrderedCollection{
			Context:    "https://www.w3.org/ns/activitystreams",
			ID:         outboxURL,
			Type:       "OrderedCollection",
			TotalItems: totalItems,
			First:      fmt.Sprintf("%s?page=1", outboxURL),
		}

		w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
		json.NewEncoder(w).Encode(collection)
		return
	}

	// Return OrderedCollectionPage
	// TODO: Implement pagination and fetch actual posts
	collectionPage := models.OrderedCollectionPage{
		Context:      "https://www.w3.org/ns/activitystreams",
		ID:           fmt.Sprintf("%s?page=%s", outboxURL, page),
		Type:         "OrderedCollectionPage",
		PartOf:       outboxURL,
		OrderedItems: []any{},
		TotalItems:   0,
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(collectionPage)
}

// Followers handles followers collection requests (/users/{username}/followers)
func (h *ActivityPubHandler) Followers(w http.ResponseWriter, r *http.Request) {
	// Extract username from URL path
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "followers" {
		http.Error(w, "Invalid followers path", http.StatusBadRequest)
		return
	}
	username := parts[0]

	// Look up user
	ctx := r.Context()
	var userID int
	err := h.db.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Count followers
	var totalItems int
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM followers WHERE user_id = $1 AND accepted = true", userID).Scan(&totalItems)

	actorID := fmt.Sprintf("%s/users/%s", h.config.Server.BaseURL, username)

	collection := models.OrderedCollection{
		Context:    "https://www.w3.org/ns/activitystreams",
		ID:         fmt.Sprintf("%s/followers", actorID),
		Type:       "OrderedCollection",
		TotalItems: totalItems,
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(collection)
}

// Following handles following collection requests (/users/{username}/following)
func (h *ActivityPubHandler) Following(w http.ResponseWriter, r *http.Request) {
	// Extract username from URL path
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "following" {
		http.Error(w, "Invalid following path", http.StatusBadRequest)
		return
	}
	username := parts[0]

	// Look up user
	ctx := r.Context()
	var userID int
	err := h.db.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Count following
	var totalItems int
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM following WHERE user_id = $1 AND accepted = true", userID).Scan(&totalItems)

	actorID := fmt.Sprintf("%s/users/%s", h.config.Server.BaseURL, username)

	collection := models.OrderedCollection{
		Context:    "https://www.w3.org/ns/activitystreams",
		ID:         fmt.Sprintf("%s/following", actorID),
		Type:       "OrderedCollection",
		TotalItems: totalItems,
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(collection)
}
