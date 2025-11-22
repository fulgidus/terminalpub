package models

import "time"

// SSHKey represents an SSH public key associated with a user
type SSHKey struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	PublicKey   string     `json:"public_key"`   // Full SSH public key
	Fingerprint string     `json:"fingerprint"`  // SHA256 fingerprint
	KeyType     string     `json:"key_type"`     // e.g., "ssh-rsa", "ssh-ed25519"
	Comment     string     `json:"comment"`      // Optional comment from key
	LastUsedAt  *time.Time `json:"last_used_at"` // Last time this key was used
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
