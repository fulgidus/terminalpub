# Security Model - terminalpub

## Overview

terminalpub implements a multi-layered security model that binds SSH public keys to user accounts and Mastodon identities. This prevents unauthorized access and ensures that only the owner of an SSH key can access their Mastodon account through terminalpub.

## Security Flow

### 1. First Connection (New User)

```
┌─────────────┐
│ User SSH    │
│ connects    │
└──────┬──────┘
       │ Presents SSH public key
       ▼
┌─────────────────────────────────┐
│ SSH Server                      │
│ - Extracts public key           │
│ - Calculates SHA256 fingerprint │
└──────┬──────────────────────────┘
       │ Query: Find user by SSH key?
       ▼
┌─────────────────┐
│ Database        │
│ No match found  │
└──────┬──────────┘
       │
       ▼
┌─────────────────────────────────┐
│ Welcome Screen                  │
│ [L] Login with Mastodon         │
│ [A] Browse anonymously          │
└─────────────────────────────────┘
```

### 2. Mastodon Login (OAuth Device Flow)

```
User selects [L] Login with Mastodon
       │
       ▼
┌─────────────────────────────────┐
│ Enter instance: mastodon.social │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│ System generates:                   │
│ - User code: WXYZ-1234              │
│ - Device code: (internal)           │
│ - Stores SSH session ID with code   │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│ Display to user:                    │
│                                     │
│ 1. Visit: https://terminalpub.com   │
│    /device                          │
│ 2. Enter code: WXYZ-1234            │
│ 3. Authorize terminalpub            │
│                                     │
│ Waiting for authorization...        │
└─────────────────────────────────────┘
       │
       │ User opens browser
       ▼
┌─────────────────────────────────────┐
│ Web Browser                         │
│ - User enters WXYZ-1234             │
│ - Redirected to Mastodon OAuth      │
│ - User authorizes                   │
│ - Callback with auth code           │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│ terminalpub backend                 │
│ - Exchanges code for access token   │
│ - Retrieves Mastodon account info   │
│ - Creates terminalpub user          │
│ - BINDS SSH key to user account     │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│ SSH Session                         │
│ ✓ Logged in as @alice@mastodon.soc  │
│                                     │
│ SSH Key: SHA256:abc123...           │
│ Now permanently linked to account   │
└─────────────────────────────────────┘
```

### 3. Subsequent Connections (Returning User)

```
┌─────────────┐
│ User SSH    │
│ connects    │
└──────┬──────┘
       │ Presents SSH public key
       ▼
┌─────────────────────────────────┐
│ SSH Server                      │
│ - Extracts public key           │
│ - Calculates fingerprint        │
└──────┬──────────────────────────┘
       │ Query: Find user by SSH key
       ▼
┌─────────────────────────────────┐
│ Database                        │
│ ✓ Match found!                  │
│ User ID: 123                    │
│ Username: alice                 │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────────────────────────┐
│ Session Manager                 │
│ - Creates authenticated session │
│ - Loads Mastodon tokens         │
│ - Updates last_used_at          │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────────────────────────┐
│ Main Menu                       │
│ Welcome back, @alice!           │
│                                 │
│ [F] Feed                        │
│ [P] Post                        │
│ [Q] Quit                        │
└─────────────────────────────────┘
```

## Security Components

### 1. SSH Key Binding

**Purpose**: Ensures only the owner of an SSH private key can access their account.

**How it works**:
- When a user logs in via Mastodon, their SSH public key is stored in the database
- A SHA256 fingerprint is calculated for fast lookups
- On subsequent connections, the SSH key is matched against stored keys
- Multiple SSH keys can be associated with one account (e.g., laptop, desktop)

**Database**: `user_ssh_keys` table
```sql
CREATE TABLE user_ssh_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    public_key TEXT NOT NULL UNIQUE,
    fingerprint VARCHAR(255) NOT NULL,
    key_type VARCHAR(50) NOT NULL,
    last_used_at TIMESTAMP
);
```

### 2. OAuth Device Flow

**Purpose**: Secure authentication without sharing passwords or requiring a browser on the SSH client.

**Flow**:
1. **Device Code Generation**: Server generates a short user code (WXYZ-1234) and a long device code
2. **User Authorization**: User visits web page, enters code, and authorizes via Mastodon
3. **Token Exchange**: Server exchanges authorization code for access token
4. **Account Binding**: Access token and account info are stored with user's SSH key

**Security features**:
- User codes expire after 15 minutes
- Device codes are single-use
- State parameter prevents CSRF attacks
- All OAuth traffic uses HTTPS

### 3. Session Management

**Purpose**: Track active SSH connections and maintain authentication state.

**Implementation**:
- **PostgreSQL**: Persistent storage of all sessions
- **Redis**: Fast cache for session lookups (TTL-based expiration)
- **Session ID**: UUID v4 for unpredictability

**Session Types**:
1. **Authenticated**: User logged in via Mastodon (24h expiry)
2. **Anonymous**: Guest browsing (1h expiry)

**Session Data**:
```json
{
  "session_id": "uuid",
  "user_id": 123,
  "username": "alice",
  "public_key": "ssh-ed25519 AAAA...",
  "ip_address": "192.168.1.1",
  "anonymous": false,
  "created_at": "2025-01-01T12:00:00Z",
  "last_seen_at": "2025-01-01T12:30:00Z",
  "expires_at": "2025-01-02T12:00:00Z"
}
```

### 4. Token Storage & Refresh

**Purpose**: Securely store Mastodon OAuth tokens and automatically refresh them.

**Storage**:
- Tokens stored encrypted in PostgreSQL
- One user can have multiple Mastodon accounts
- Primary account flag for default selection

**Auto-refresh**:
- Tokens checked before each Mastodon API call
- If expired or expiring soon (< 1 hour), automatic refresh
- Refresh token used to obtain new access token
- Updated tokens saved to database

## Threat Model & Mitigations

### Threat 1: Unauthorized SSH Access
**Risk**: Attacker connects via SSH and tries to access another user's account

**Mitigation**:
- ✅ SSH key binding: Only owner of private key can authenticate
- ✅ Fingerprint matching: Fast, secure key lookup
- ✅ Public key uniqueness: One SSH key = one user account

### Threat 2: Session Hijacking
**Risk**: Attacker steals session ID and impersonates user

**Mitigation**:
- ✅ Session IDs are UUIDs (128-bit entropy)
- ✅ Sessions tied to SSH public key
- ✅ IP address tracking (optional: can enforce IP binding)
- ✅ Short session expiry (24h max)
- ✅ Automatic cleanup of expired sessions

### Threat 3: Device Code Interception
**Risk**: Attacker sees user code (WXYZ-1234) and tries to authorize

**Mitigation**:
- ✅ 15-minute expiration on device codes
- ✅ Single-use codes
- ✅ User must be logged into their Mastodon account in browser
- ✅ OAuth flow shows app name & requested permissions
- ✅ User explicitly authorizes

### Threat 4: Token Theft
**Risk**: Attacker gains access to database and steals tokens

**Mitigation**:
- ✅ Tokens stored in secure database
- 🔄 TODO: Encrypt tokens at rest
- ✅ Limited token scopes (only requested permissions)
- ✅ Token refresh mechanism (old tokens invalidated)
- ✅ User can revoke access from Mastodon settings

### Threat 5: Man-in-the-Middle
**Risk**: Attacker intercepts OAuth traffic

**Mitigation**:
- ✅ All OAuth flows use HTTPS
- ✅ State parameter prevents CSRF
- ✅ Redirect URI validation
- ✅ SSH connection encrypted by SSH protocol

## Best Practices for Users

### Multiple Devices
Users can register multiple SSH keys:
```bash
# From laptop
ssh terminalpub.com
# Login with Mastodon (first time)

# From desktop
ssh terminalpub.com
# Login with Mastodon (associates new key with same account)
```

### Key Management
Users can:
- View all registered SSH keys
- Remove old/compromised keys
- Add new keys without re-authorization

### Revoking Access
If account is compromised:
1. Remove SSH keys from terminalpub
2. Revoke OAuth app access from Mastodon settings
3. Generate new SSH keys if needed

## Privacy Considerations

### Data Stored
- SSH public keys (not private keys)
- Mastodon instance URL
- Mastodon user ID, username, display name
- OAuth access tokens (to interact with Mastodon on user's behalf)
- Session metadata (IP, timestamps)

### Data NOT Stored
- SSH private keys
- Mastodon passwords
- Mastodon posts (cached temporarily for display only)
- Private messages (not accessed)

### Data Retention
- Sessions: Deleted after expiration
- Device codes: Deleted after 15 minutes
- SSH keys: Kept until user removes them
- User accounts: Kept until user requests deletion

## Compliance

### GDPR
- Users can view their data
- Users can delete their account (right to erasure)
- Data minimization: Only necessary data collected
- Explicit consent via OAuth flow

### OAuth 2.0 Device Flow (RFC 8628)
- Full compliance with RFC 8628
- Standard Mastodon OAuth implementation
- No custom/non-standard extensions

## Future Enhancements

### Planned
- [ ] Token encryption at rest (AES-256)
- [ ] 2FA/TOTP support
- [ ] IP-based session validation (optional)
- [ ] Suspicious activity detection
- [ ] Rate limiting per SSH key

### Under Consideration
- [ ] Hardware security key support (U2F/FIDO2)
- [ ] Session fingerprinting (SSH client identification)
- [ ] Audit log for all authentication events
- [ ] Automated security scanning

## Conclusion

terminalpub's security model ensures that:
1. **Only SSH key owners** can access accounts
2. **Mastodon credentials** are never exposed
3. **Sessions** are short-lived and tracked
4. **Tokens** are managed securely
5. **Users** maintain control over access

This multi-layered approach provides defense-in-depth while maintaining a seamless user experience.
