package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Post represents a user's post/status
type Post struct {
	ID          int             `json:"id" db:"id"`
	UserID      int             `json:"user_id" db:"user_id"`
	Content     string          `json:"content" db:"content"`
	ContentType string          `json:"content_type" db:"content_type"`
	InReplyToID *int            `json:"in_reply_to_id,omitempty" db:"in_reply_to_id"`
	Visibility  string          `json:"visibility" db:"visibility"`
	PublishedAt time.Time       `json:"published_at" db:"published_at"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	APID        string          `json:"ap_id,omitempty" db:"ap_id"`
	APType      string          `json:"ap_type" db:"ap_type"`
	APObject    json.RawMessage `json:"ap_object,omitempty" db:"ap_object"`
}

// Follower represents someone following a user
type Follower struct {
	ID                  int       `json:"id" db:"id"`
	UserID              int       `json:"user_id" db:"user_id"`
	FollowerActorID     string    `json:"follower_actor_id" db:"follower_actor_id"`
	FollowerUsername    string    `json:"follower_username" db:"follower_username"`
	FollowerInbox       string    `json:"follower_inbox" db:"follower_inbox"`
	FollowerSharedInbox string    `json:"follower_shared_inbox,omitempty" db:"follower_shared_inbox"`
	Accepted            bool      `json:"accepted" db:"accepted"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// Following represents a user following someone
type Following struct {
	ID                int       `json:"id" db:"id"`
	UserID            int       `json:"user_id" db:"user_id"`
	TargetActorID     string    `json:"target_actor_id" db:"target_actor_id"`
	TargetUsername    string    `json:"target_username" db:"target_username"`
	TargetInbox       string    `json:"target_inbox" db:"target_inbox"`
	TargetSharedInbox string    `json:"target_shared_inbox,omitempty" db:"target_shared_inbox"`
	Accepted          bool      `json:"accepted" db:"accepted"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Activity represents an ActivityPub activity
type Activity struct {
	ID           int             `json:"id" db:"id"`
	UserID       *int            `json:"user_id,omitempty" db:"user_id"`
	ActivityType string          `json:"activity_type" db:"activity_type"`
	ActorID      string          `json:"actor_id" db:"actor_id"`
	ObjectID     string          `json:"object_id,omitempty" db:"object_id"`
	TargetID     string          `json:"target_id,omitempty" db:"target_id"`
	ActivityJSON json.RawMessage `json:"activity_json" db:"activity_json"`
	Direction    string          `json:"direction" db:"direction"` // "inbound" or "outbound"
	Processed    bool            `json:"processed" db:"processed"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// Like represents a like on a post
type Like struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	PostID    *int      `json:"post_id,omitempty" db:"post_id"`
	ActorID   string    `json:"actor_id" db:"actor_id"`
	APID      string    `json:"ap_id,omitempty" db:"ap_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Boost represents a boost/reblog of a post
type Boost struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	PostID    *int      `json:"post_id,omitempty" db:"post_id"`
	ActorID   string    `json:"actor_id" db:"actor_id"`
	APID      string    `json:"ap_id,omitempty" db:"ap_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Actor represents an ActivityPub Actor (Person)
type Actor struct {
	Context                   any            `json:"@context"`
	ID                        string         `json:"id"`
	Type                      string         `json:"type"`
	PreferredUsername         string         `json:"preferredUsername"`
	Name                      string         `json:"name,omitempty"`
	Summary                   string         `json:"summary,omitempty"`
	Inbox                     string         `json:"inbox"`
	Outbox                    string         `json:"outbox"`
	Followers                 string         `json:"followers"`
	Following                 string         `json:"following"`
	PublicKey                 ActorPublicKey `json:"publicKey"`
	Endpoints                 map[string]any `json:"endpoints,omitempty"`
	URL                       string         `json:"url,omitempty"`
	ManuallyApprovesFollowers bool           `json:"manuallyApprovesFollowers"`
	Published                 string         `json:"published,omitempty"`
}

// ActorPublicKey represents the public key in an Actor object
type ActorPublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

// APNote represents an ActivityPub Note object
type APNote struct {
	Context      any               `json:"@context,omitempty"`
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	AttributedTo string            `json:"attributedTo"`
	Content      string            `json:"content"`
	ContentMap   map[string]string `json:"contentMap,omitempty"`
	Published    string            `json:"published"`
	To           []string          `json:"to,omitempty"`
	CC           []string          `json:"cc,omitempty"`
	InReplyTo    string            `json:"inReplyTo,omitempty"`
	Tag          []any             `json:"tag,omitempty"`
	Attachment   []any             `json:"attachment,omitempty"`
	Sensitive    bool              `json:"sensitive,omitempty"`
}

// APActivity represents a generic ActivityPub Activity
type APActivity struct {
	Context   any      `json:"@context,omitempty"`
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Actor     string   `json:"actor"`
	Object    any      `json:"object,omitempty"`
	Target    string   `json:"target,omitempty"`
	To        []string `json:"to,omitempty"`
	CC        []string `json:"cc,omitempty"`
	Published string   `json:"published,omitempty"`
}

// OrderedCollection represents an ActivityPub OrderedCollection
type OrderedCollection struct {
	Context    any    `json:"@context,omitempty"`
	ID         string `json:"id"`
	Type       string `json:"type"`
	TotalItems int    `json:"totalItems"`
	First      string `json:"first,omitempty"`
	Last       string `json:"last,omitempty"`
}

// OrderedCollectionPage represents a page in an OrderedCollection
type OrderedCollectionPage struct {
	Context      any    `json:"@context,omitempty"`
	ID           string `json:"id"`
	Type         string `json:"type"`
	PartOf       string `json:"partOf"`
	TotalItems   int    `json:"totalItems"`
	OrderedItems []any  `json:"orderedItems"`
	Next         string `json:"next,omitempty"`
	Prev         string `json:"prev,omitempty"`
}

// Value implements driver.Valuer for json.RawMessage in Activity
func (a Activity) Value() (driver.Value, error) {
	return json.Marshal(a.ActivityJSON)
}

// Scan implements sql.Scanner for json.RawMessage in Activity
func (a *Activity) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &a.ActivityJSON)
}
