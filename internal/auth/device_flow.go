package auth

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/fulgidus/terminalpub/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DeviceCodeLength is the length of the device code
	DeviceCodeLength = 32

	// UserCodeLength is the length of the user-facing code (8 chars + hyphen)
	UserCodeLength = 8

	// DeviceCodeExpiry is how long a device code is valid
	DeviceCodeExpiry = 15 * time.Minute
)

// DeviceFlowService handles OAuth Device Flow operations
type DeviceFlowService struct {
	db              *pgxpool.Pool
	verificationURI string
}

// NewDeviceFlowService creates a new DeviceFlowService instance
func NewDeviceFlowService(db *pgxpool.Pool, verificationURI string) *DeviceFlowService {
	return &DeviceFlowService{
		db:              db,
		verificationURI: verificationURI,
	}
}

// DeviceAuthResponse contains the information shown to the user
type DeviceAuthResponse struct {
	UserCode        string    `json:"user_code"`
	DeviceCode      string    `json:"device_code"`
	VerificationURI string    `json:"verification_uri"`
	ExpiresIn       int       `json:"expires_in"`
	Interval        int       `json:"interval"`
	ExpiresAt       time.Time `json:"-"`
}

// generateRandomCode generates a cryptographically random code
func generateRandomCode(length int) (string, error) {
	// Generate random bytes (use more bytes for better randomness)
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base32 without padding
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)

	// Take first `length` characters
	if len(encoded) < length {
		return encoded, nil
	}
	return encoded[:length], nil
}

// generateUserCode generates a user-friendly 8-character code (XXXX-XXXX format)
func generateUserCode() (string, error) {
	code, err := generateRandomCode(UserCodeLength)
	if err != nil {
		return "", err
	}

	// Format as XXXX-XXXX for readability
	if len(code) >= 8 {
		return fmt.Sprintf("%s-%s", code[:4], code[4:8]), nil
	}

	return code, nil
}

// generateDeviceCode generates a long device code for polling
func generateDeviceCode() (string, error) {
	return generateRandomCode(DeviceCodeLength)
}

// InitiateDeviceFlow starts a new device authorization flow
func (d *DeviceFlowService) InitiateDeviceFlow(ctx context.Context, instanceURL, sshSessionID string) (*DeviceAuthResponse, error) {
	// Generate codes
	userCode, err := generateUserCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user code: %w", err)
	}

	deviceCode, err := generateDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device code: %w", err)
	}

	expiresAt := time.Now().Add(DeviceCodeExpiry)

	// Store in database
	query := `
		INSERT INTO device_codes (user_code, device_code, instance_url, ssh_session_id, verification_uri, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int
	err = d.db.QueryRow(ctx, query,
		userCode,
		deviceCode,
		NormalizeInstanceURL(instanceURL),
		sshSessionID,
		d.verificationURI,
		expiresAt,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to store device code: %w", err)
	}

	return &DeviceAuthResponse{
		UserCode:        userCode,
		DeviceCode:      deviceCode,
		VerificationURI: d.verificationURI,
		ExpiresIn:       int(DeviceCodeExpiry.Seconds()),
		Interval:        5, // Poll every 5 seconds
		ExpiresAt:       expiresAt,
	}, nil
}

// GetDeviceCodeByUserCode retrieves a device code by user code
func (d *DeviceFlowService) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*models.DeviceCode, error) {
	// Normalize user code (remove spaces, convert to uppercase)
	userCode = strings.ToUpper(strings.ReplaceAll(userCode, " ", ""))

	query := `
		SELECT id, user_code, device_code, instance_url, ssh_session_id, 
		       verification_uri, expires_at, authorized, user_id, created_at
		FROM device_codes
		WHERE user_code = $1
	`

	var dc models.DeviceCode
	err := d.db.QueryRow(ctx, query, userCode).Scan(
		&dc.ID,
		&dc.UserCode,
		&dc.DeviceCode,
		&dc.InstanceURL,
		&dc.SSHSessionID,
		&dc.VerificationURI,
		&dc.ExpiresAt,
		&dc.Authorized,
		&dc.UserID,
		&dc.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("device code not found: %w", err)
	}

	// Check if expired
	if time.Now().After(dc.ExpiresAt) {
		return nil, fmt.Errorf("device code expired")
	}

	return &dc, nil
}

// GetDeviceCodeByDeviceCode retrieves a device code by device code (for polling)
func (d *DeviceFlowService) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*models.DeviceCode, error) {
	query := `
		SELECT id, user_code, device_code, instance_url, ssh_session_id, 
		       verification_uri, expires_at, authorized, user_id, created_at
		FROM device_codes
		WHERE device_code = $1
	`

	var dc models.DeviceCode
	err := d.db.QueryRow(ctx, query, deviceCode).Scan(
		&dc.ID,
		&dc.UserCode,
		&dc.DeviceCode,
		&dc.InstanceURL,
		&dc.SSHSessionID,
		&dc.VerificationURI,
		&dc.ExpiresAt,
		&dc.Authorized,
		&dc.UserID,
		&dc.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("device code not found: %w", err)
	}

	// Check if expired
	if time.Now().After(dc.ExpiresAt) {
		return nil, fmt.Errorf("device code expired")
	}

	return &dc, nil
}

// AuthorizeDeviceCode marks a device code as authorized
func (d *DeviceFlowService) AuthorizeDeviceCode(ctx context.Context, userCode string, userID int) error {
	// Normalize user code
	userCode = strings.ToUpper(strings.ReplaceAll(userCode, " ", ""))

	query := `
		UPDATE device_codes
		SET authorized = TRUE, user_id = $1
		WHERE user_code = $2 AND authorized = FALSE AND expires_at > NOW()
	`

	result, err := d.db.Exec(ctx, query, userID, userCode)
	if err != nil {
		return fmt.Errorf("failed to authorize device code: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("device code not found or already authorized")
	}

	return nil
}

// PollDeviceCode checks if a device code has been authorized (for SSH client polling)
func (d *DeviceFlowService) PollDeviceCode(ctx context.Context, deviceCode string) (bool, int, error) {
	dc, err := d.GetDeviceCodeByDeviceCode(ctx, deviceCode)
	if err != nil {
		return false, 0, err
	}

	if dc.Authorized && dc.UserID != nil {
		return true, *dc.UserID, nil
	}

	return false, 0, nil
}

// CleanupExpiredCodes removes expired device codes (should be run periodically)
func (d *DeviceFlowService) CleanupExpiredCodes(ctx context.Context) error {
	query := `DELETE FROM device_codes WHERE expires_at < NOW()`

	_, err := d.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired codes: %w", err)
	}

	return nil
}
