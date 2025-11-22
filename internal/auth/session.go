package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	// SessionExpiry is the default session expiration time
	SessionExpiry = 24 * time.Hour

	// AnonymousSessionExpiry is expiration for anonymous sessions
	AnonymousSessionExpiry = 1 * time.Hour

	// RedisSessionPrefix is the prefix for session keys in Redis
	RedisSessionPrefix = "session:"
)

// SessionManager manages SSH sessions using Redis for fast access and PostgreSQL for persistence
type SessionManager struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewSessionManager creates a new SessionManager instance
func NewSessionManager(db *pgxpool.Pool, redisClient *redis.Client) *SessionManager {
	return &SessionManager{
		db:    db,
		redis: redisClient,
	}
}

// SessionData contains cached session information
type SessionData struct {
	SessionID  string    `json:"session_id"`
	UserID     *int      `json:"user_id"`
	Username   string    `json:"username,omitempty"`
	PublicKey  string    `json:"public_key"`
	IPAddress  string    `json:"ip_address"`
	Anonymous  bool      `json:"anonymous"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CreateSession creates a new SSH session
func (sm *SessionManager) CreateSession(ctx context.Context, publicKey, ipAddress string, userID *int, anonymous bool) (*SessionData, error) {
	sessionID := uuid.New().String()

	var expiry time.Duration
	if anonymous {
		expiry = AnonymousSessionExpiry
	} else {
		expiry = SessionExpiry
	}

	now := time.Now()
	expiresAt := now.Add(expiry)

	// Store in PostgreSQL
	query := `
		INSERT INTO sessions (id, user_id, public_key, ip_address, anonymous, created_at, last_seen_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := sm.db.Exec(ctx, query,
		sessionID,
		userID,
		publicKey,
		ipAddress,
		anonymous,
		now,
		now,
		expiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create session in database: %w", err)
	}

	// Get username if authenticated
	var username string
	if userID != nil {
		err := sm.db.QueryRow(ctx, "SELECT username FROM users WHERE id = $1", *userID).Scan(&username)
		if err != nil {
			username = ""
		}
	}

	sessionData := &SessionData{
		SessionID:  sessionID,
		UserID:     userID,
		Username:   username,
		PublicKey:  publicKey,
		IPAddress:  ipAddress,
		Anonymous:  anonymous,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  expiresAt,
	}

	// Cache in Redis
	if err := sm.cacheSession(ctx, sessionData); err != nil {
		// Log error but don't fail - session is already in PostgreSQL
		fmt.Printf("warning: failed to cache session in Redis: %v\n", err)
	}

	return sessionData, nil
}

// GetSession retrieves a session by ID (tries Redis first, falls back to PostgreSQL)
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	// Try Redis first (fast path)
	sessionData, err := sm.getSessionFromRedis(ctx, sessionID)
	if err == nil {
		// Update last_seen_at
		go sm.UpdateLastSeen(context.Background(), sessionID)
		return sessionData, nil
	}

	// Fall back to PostgreSQL
	sessionData, err = sm.getSessionFromDB(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Re-cache in Redis
	_ = sm.cacheSession(ctx, sessionData)

	// Update last_seen_at
	go sm.UpdateLastSeen(context.Background(), sessionID)

	return sessionData, nil
}

// getSessionFromRedis retrieves session from Redis cache
func (sm *SessionManager) getSessionFromRedis(ctx context.Context, sessionID string) (*SessionData, error) {
	key := RedisSessionPrefix + sessionID

	data, err := sm.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session not in cache: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check expiration
	if time.Now().After(sessionData.ExpiresAt) {
		sm.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &sessionData, nil
}

// getSessionFromDB retrieves session from PostgreSQL
func (sm *SessionManager) getSessionFromDB(ctx context.Context, sessionID string) (*SessionData, error) {
	query := `
		SELECT s.id, s.user_id, s.public_key, s.ip_address, s.anonymous,
		       s.created_at, s.last_seen_at, s.expires_at, u.username
		FROM sessions s
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.id = $1 AND s.expires_at > NOW()
	`

	var sessionData SessionData
	var username *string

	err := sm.db.QueryRow(ctx, query, sessionID).Scan(
		&sessionData.SessionID,
		&sessionData.UserID,
		&sessionData.PublicKey,
		&sessionData.IPAddress,
		&sessionData.Anonymous,
		&sessionData.CreatedAt,
		&sessionData.LastSeenAt,
		&sessionData.ExpiresAt,
		&username,
	)

	if err != nil {
		return nil, fmt.Errorf("session not found in database: %w", err)
	}

	if username != nil {
		sessionData.Username = *username
	}

	return &sessionData, nil
}

// cacheSession stores session data in Redis
func (sm *SessionManager) cacheSession(ctx context.Context, sessionData *SessionData) error {
	key := RedisSessionPrefix + sessionData.SessionID

	data, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := time.Until(sessionData.ExpiresAt)
	if ttl <= 0 {
		return nil // Already expired
	}

	return sm.redis.Set(ctx, key, data, ttl).Err()
}

// UpdateLastSeen updates the last_seen_at timestamp for a session
func (sm *SessionManager) UpdateLastSeen(ctx context.Context, sessionID string) error {
	// Update in PostgreSQL
	_, err := sm.db.Exec(ctx,
		"UPDATE sessions SET last_seen_at = NOW() WHERE id = $1",
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("failed to update last_seen_at: %w", err)
	}

	// Update in Redis cache if exists
	sessionData, err := sm.getSessionFromRedis(ctx, sessionID)
	if err == nil {
		sessionData.LastSeenAt = time.Now()
		_ = sm.cacheSession(ctx, sessionData)
	}

	return nil
}

// DeleteSession deletes a session from both Redis and PostgreSQL
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete from Redis
	key := RedisSessionPrefix + sessionID
	_ = sm.redis.Del(ctx, key).Err()

	// Delete from PostgreSQL
	_, err := sm.db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// UpgradeSessionToAuthenticated upgrades an anonymous session to authenticated
func (sm *SessionManager) UpgradeSessionToAuthenticated(ctx context.Context, sessionID string, userID int) error {
	// Update in PostgreSQL
	query := `
		UPDATE sessions
		SET user_id = $1, anonymous = FALSE, expires_at = NOW() + INTERVAL '24 hours'
		WHERE id = $2
	`

	result, err := sm.db.Exec(ctx, query, userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to upgrade session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}

	// Delete from Redis to force refresh
	key := RedisSessionPrefix + sessionID
	_ = sm.redis.Del(ctx, key).Err()

	return nil
}

// CleanupExpiredSessions removes expired sessions (should be run periodically)
func (sm *SessionManager) CleanupExpiredSessions(ctx context.Context) error {
	// Clean from PostgreSQL
	_, err := sm.db.Exec(ctx, "DELETE FROM sessions WHERE expires_at < NOW()")
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	// Redis sessions will expire automatically via TTL

	return nil
}

// ListUserSessions lists all active sessions for a user
func (sm *SessionManager) ListUserSessions(ctx context.Context, userID int) ([]SessionData, error) {
	query := `
		SELECT s.id, s.user_id, s.public_key, s.ip_address, s.anonymous,
		       s.created_at, s.last_seen_at, s.expires_at, u.username
		FROM sessions s
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.user_id = $1 AND s.expires_at > NOW()
		ORDER BY s.last_seen_at DESC
	`

	rows, err := sm.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionData
	for rows.Next() {
		var session SessionData
		var username *string

		err := rows.Scan(
			&session.SessionID,
			&session.UserID,
			&session.PublicKey,
			&session.IPAddress,
			&session.Anonymous,
			&session.CreatedAt,
			&session.LastSeenAt,
			&session.ExpiresAt,
			&username,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if username != nil {
			session.Username = *username
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}
