package models

import "time"

// User represents a terminalpub user
type User struct {
	ID                      int       `json:"id"`
	Username                string    `json:"username"`
	Email                   string    `json:"email,omitempty"`
	PasswordHash            string    `json:"-"`
	PrimaryMastodonInstance string    `json:"primary_mastodon_instance,omitempty"`
	PrimaryMastodonID       string    `json:"primary_mastodon_id,omitempty"`
	PrimaryMastodonAcct     string    `json:"primary_mastodon_acct,omitempty"`
	PrivateKey              string    `json:"-"`
	PublicKey               string    `json:"public_key,omitempty"`
	ActorURL                string    `json:"actor_url,omitempty"`
	InboxURL                string    `json:"inbox_url,omitempty"`
	OutboxURL               string    `json:"outbox_url,omitempty"`
	FollowersURL            string    `json:"followers_url,omitempty"`
	FollowingURL            string    `json:"following_url,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	Bio                     string    `json:"bio,omitempty"`
	AvatarURL               string    `json:"avatar_url,omitempty"`
}
