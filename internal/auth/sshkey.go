package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"
)

// SSHKeyService manages SSH public keys for users
type SSHKeyService struct {
	db *pgxpool.Pool
}

// NewSSHKeyService creates a new SSHKeyService instance
func NewSSHKeyService(db *pgxpool.Pool) *SSHKeyService {
	return &SSHKeyService{db: db}
}

// ParseSSHPublicKey parses an SSH public key and extracts metadata
func ParseSSHPublicKey(publicKeyStr string) (*models.SSHKey, error) {
	publicKeyStr = strings.TrimSpace(publicKeyStr)

	// Parse the SSH public key
	publicKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKeyStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH public key: %w", err)
	}

	// Calculate fingerprint (SHA256)
	fingerprint := calculateFingerprint(publicKey)

	// Get key type
	keyType := publicKey.Type()

	return &models.SSHKey{
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		KeyType:     keyType,
		Comment:     comment,
	}, nil
}

// calculateFingerprint generates SHA256 fingerprint for an SSH key
func calculateFingerprint(publicKey ssh.PublicKey) string {
	hash := sha256.Sum256(publicKey.Marshal())
	b64 := base64.RawStdEncoding.EncodeToString(hash[:])
	return "SHA256:" + b64
}

// GetUserBySSHKey finds a user by their SSH public key
func (s *SSHKeyService) GetUserBySSHKey(ctx context.Context, publicKeyStr string) (*models.User, error) {
	// Parse the key to get fingerprint
	keyInfo, err := ParseSSHPublicKey(publicKeyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH key: %w", err)
	}

	// Look up user by fingerprint (faster) or public key
	query := `
		SELECT u.id, u.username, u.email, COALESCE(u.password_hash, ''), COALESCE(u.primary_mastodon_instance, ''),
		       COALESCE(u.primary_mastodon_id, ''), COALESCE(u.primary_mastodon_acct, ''), u.private_key, u.public_key,
		       u.actor_url, u.inbox_url, u.outbox_url, u.followers_url, u.following_url,
		       u.created_at, u.updated_at, COALESCE(u.bio, ''), COALESCE(u.avatar_url, '')
		FROM users u
		INNER JOIN user_ssh_keys k ON k.user_id = u.id
		WHERE k.fingerprint = $1 OR k.public_key = $2
		LIMIT 1
	`

	var user models.User
	err = s.db.QueryRow(ctx, query, keyInfo.Fingerprint, publicKeyStr).Scan(
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
		return nil, fmt.Errorf("user not found for SSH key: %w", err)
	}

	// Update last_used_at for this key
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.db.Exec(updateCtx,
			"UPDATE user_ssh_keys SET last_used_at = NOW() WHERE fingerprint = $1",
			keyInfo.Fingerprint,
		)
	}()

	return &user, nil
}

// AddSSHKeyToUser associates an SSH key with a user
func (s *SSHKeyService) AddSSHKeyToUser(ctx context.Context, userID int, publicKeyStr string) (*models.SSHKey, error) {
	// Parse the key
	keyInfo, err := ParseSSHPublicKey(publicKeyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH key: %w", err)
	}

	// Check if key already exists (global uniqueness)
	var existingUserID int
	err = s.db.QueryRow(ctx,
		"SELECT user_id FROM user_ssh_keys WHERE fingerprint = $1 OR public_key = $2",
		keyInfo.Fingerprint, publicKeyStr,
	).Scan(&existingUserID)

	if err == nil {
		if existingUserID == userID {
			return nil, fmt.Errorf("SSH key already associated with this user")
		}
		return nil, fmt.Errorf("SSH key already associated with another user")
	}

	// Insert new key
	query := `
		INSERT INTO user_ssh_keys (user_id, public_key, fingerprint, key_type, comment, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRow(ctx, query,
		userID,
		keyInfo.PublicKey,
		keyInfo.Fingerprint,
		keyInfo.KeyType,
		keyInfo.Comment,
	).Scan(&keyInfo.ID, &keyInfo.CreatedAt, &keyInfo.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to add SSH key: %w", err)
	}

	keyInfo.UserID = userID
	now := time.Now()
	keyInfo.LastUsedAt = &now

	return keyInfo, nil
}

// RemoveSSHKey removes an SSH key from a user
func (s *SSHKeyService) RemoveSSHKey(ctx context.Context, userID int, keyID int) error {
	result, err := s.db.Exec(ctx,
		"DELETE FROM user_ssh_keys WHERE id = $1 AND user_id = $2",
		keyID, userID,
	)

	if err != nil {
		return fmt.Errorf("failed to remove SSH key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("SSH key not found")
	}

	return nil
}

// ListUserSSHKeys lists all SSH keys for a user
func (s *SSHKeyService) ListUserSSHKeys(ctx context.Context, userID int) ([]models.SSHKey, error) {
	query := `
		SELECT id, user_id, public_key, fingerprint, key_type, comment, last_used_at, created_at, updated_at
		FROM user_ssh_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}
	defer rows.Close()

	var keys []models.SSHKey
	for rows.Next() {
		var key models.SSHKey
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.PublicKey,
			&key.Fingerprint,
			&key.KeyType,
			&key.Comment,
			&key.LastUsedAt,
			&key.CreatedAt,
			&key.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan SSH key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}
