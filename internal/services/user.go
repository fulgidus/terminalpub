package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserService handles user-related operations
type UserService struct {
	db *pgxpool.Pool
}

// NewUserService creates a new UserService instance
func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new terminalpub user
func (s *UserService) CreateUser(ctx context.Context, username, email string) (*models.User, error) {
	// Generate ActivityPub keypair for the user
	privateKey, publicKey, err := generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Build ActivityPub URLs (will be used in Phase 3)
	// For now we use placeholder domain
	domain := "51.91.97.241" // TODO: get from config
	actorURL := fmt.Sprintf("http://%s/users/%s", domain, username)
	inboxURL := fmt.Sprintf("http://%s/users/%s/inbox", domain, username)
	outboxURL := fmt.Sprintf("http://%s/users/%s/outbox", domain, username)
	followersURL := fmt.Sprintf("http://%s/users/%s/followers", domain, username)
	followingURL := fmt.Sprintf("http://%s/users/%s/following", domain, username)

	query := `
		INSERT INTO users (
			username, email, private_key, public_key,
			actor_url, inbox_url, outbox_url, followers_url, following_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	user := &models.User{
		Username:     username,
		Email:        email,
		PrivateKey:   privateKey,
		PublicKey:    publicKey,
		ActorURL:     actorURL,
		InboxURL:     inboxURL,
		OutboxURL:    outboxURL,
		FollowersURL: followersURL,
		FollowingURL: followingURL,
	}

	err = s.db.QueryRow(ctx, query,
		username,
		email,
		privateKey,
		publicKey,
		actorURL,
		inboxURL,
		outboxURL,
		followersURL,
		followingURL,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, primary_mastodon_instance,
		       primary_mastodon_id, primary_mastodon_acct, private_key, public_key,
		       actor_url, inbox_url, outbox_url, followers_url, following_url,
		       created_at, updated_at, bio, avatar_url
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := s.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.PrimaryMastodonInstance,
		&user.PrimaryMastodonID,
		&user.PrimaryMastodonAcct,
		&user.PrivateKey,
		&user.PublicKey,
		&user.ActorURL,
		&user.InboxURL,
		&user.OutboxURL,
		&user.FollowersURL,
		&user.FollowingURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Bio,
		&user.AvatarURL,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, primary_mastodon_instance,
		       primary_mastodon_id, primary_mastodon_acct, private_key, public_key,
		       actor_url, inbox_url, outbox_url, followers_url, following_url,
		       created_at, updated_at, bio, avatar_url
		FROM users
		WHERE username = $1
	`

	var user models.User
	err := s.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.PrimaryMastodonInstance,
		&user.PrimaryMastodonID,
		&user.PrimaryMastodonAcct,
		&user.PrivateKey,
		&user.PublicKey,
		&user.ActorURL,
		&user.InboxURL,
		&user.OutboxURL,
		&user.FollowersURL,
		&user.FollowingURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Bio,
		&user.AvatarURL,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

// GetOrCreateUser gets an existing user or creates a new one (upsert)
func (s *UserService) GetOrCreateUser(ctx context.Context, username, email string) (*models.User, error) {
	// Try to get existing user first
	user, err := s.GetUserByUsername(ctx, username)
	if err == nil {
		// User exists, return it
		return user, nil
	}

	// User doesn't exist, create new one
	// Generate ActivityPub keypair for the user
	privateKey, publicKey, err := generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Build ActivityPub URLs
	domain := "51.91.97.241" // TODO: get from config
	actorURL := fmt.Sprintf("http://%s/users/%s", domain, username)
	inboxURL := fmt.Sprintf("http://%s/users/%s/inbox", domain, username)
	outboxURL := fmt.Sprintf("http://%s/users/%s/outbox", domain, username)
	followersURL := fmt.Sprintf("http://%s/users/%s/followers", domain, username)
	followingURL := fmt.Sprintf("http://%s/users/%s/following", domain, username)

	// Use INSERT ... ON CONFLICT to handle race conditions
	query := `
		INSERT INTO users (
			username, email, private_key, public_key,
			actor_url, inbox_url, outbox_url, followers_url, following_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (username) DO UPDATE SET
			updated_at = NOW()
		RETURNING id, username, email, COALESCE(password_hash, ''), COALESCE(primary_mastodon_instance, ''),
		          COALESCE(primary_mastodon_id, ''), COALESCE(primary_mastodon_acct, ''), private_key, public_key,
		          actor_url, inbox_url, outbox_url, followers_url, following_url,
		          created_at, updated_at, COALESCE(bio, ''), COALESCE(avatar_url, '')
	`

	user = &models.User{}
	err = s.db.QueryRow(ctx, query,
		username,
		email,
		privateKey,
		publicKey,
		actorURL,
		inboxURL,
		outboxURL,
		followersURL,
		followingURL,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.PrimaryMastodonInstance,
		&user.PrimaryMastodonID,
		&user.PrimaryMastodonAcct,
		&user.PrivateKey,
		&user.PublicKey,
		&user.ActorURL,
		&user.InboxURL,
		&user.OutboxURL,
		&user.FollowersURL,
		&user.FollowingURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Bio,
		&user.AvatarURL,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create or get user: %w", err)
	}

	return user, nil
}

// UpdatePrimaryMastodonAccount updates the user's primary Mastodon account info
func (s *UserService) UpdatePrimaryMastodonAccount(ctx context.Context, userID int, instance, mastodonID, acct string) error {
	query := `
		UPDATE users
		SET primary_mastodon_instance = $1,
		    primary_mastodon_id = $2,
		    primary_mastodon_acct = $3,
		    updated_at = NOW()
		WHERE id = $4
	`

	result, err := s.db.Exec(ctx, query, instance, mastodonID, acct, userID)
	if err != nil {
		return fmt.Errorf("failed to update primary mastodon account: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// generateKeyPair generates an RSA keypair for ActivityPub
func generateKeyPair() (privateKeyPEM string, publicKeyPEM string, err error) {
	// Generate 2048-bit RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode private key to PEM
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyPEM = string(pem.EncodeToMemory(privateKeyBlock))

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyPEM = string(pem.EncodeToMemory(publicKeyBlock))

	return privateKeyPEM, publicKeyPEM, nil
}
