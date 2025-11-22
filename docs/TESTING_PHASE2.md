# Phase 2 Authentication - Manual Testing Guide

## ✅ What's Implemented

### Backend Services
- ✅ OAuth Device Flow
- ✅ Mastodon app registration (automatic per instance)
- ✅ Token management (exchange, storage, refresh)
- ✅ SSH key binding to users
- ✅ Session management (Redis + PostgreSQL)
- ✅ User service for account creation

### HTTP Endpoints
- ✅ `/device` - Device code entry form
- ✅ `/oauth/callback` - OAuth callback handler
- ✅ `/health` - Health check

### TUI Features
- ✅ Welcome screen
- ✅ Login with Mastodon flow
- ✅ Anonymous mode
- ✅ Device code display
- ✅ Polling for authorization
- ✅ SSH key auto-login (returning users)

## 🧪 How to Test the Complete Login Flow

### Test 1: First Time Login

1. **Connect via SSH**
   ```bash
   ssh 51.91.97.241
   ```

2. **You'll see the welcome screen:**
   ```
   ╔════════════════════════════════════════════╗
   ║        Welcome to terminalpub!             ║
   ║        ActivityPub for terminals           ║
   ╠════════════════════════════════════════════╣
   ║                                            ║
   ║  Connected as: guest                       ║
   ║                                            ║
   ║  Press a key to continue:                  ║
   ║                                            ║
   ║  [L] Login with Mastodon                   ║
   ║  [A] Continue anonymously                  ║
   ║  [Q] Quit                                  ║
   ║                                            ║
   ╚════════════════════════════════════════════╝
   ```

3. **Press `L` for Login**
   - The screen will ask for your Mastodon instance

4. **Enter your Mastodon instance** (e.g., `mastodon.social`)
   - Press Enter

5. **You'll see a device code screen:**
   ```
   ╔════════════════════════════════════════════╗
   ║        Waiting for Authorization           ║
   ╠════════════════════════════════════════════╣
   ║                                            ║
   ║  1. Open your browser and visit:          ║
   ║                                            ║
   ║     http://51.91.97.241/device             ║
   ║                                            ║
   ║  2. Enter this code:                       ║
   ║                                            ║
   ║     ┌────────────┐                         ║
   ║     │  WXYZ-1234 │                         ║
   ║     └────────────┘                         ║
   ║                                            ║
   ║  3. Authorize terminalpub access           ║
   ║                                            ║
   ║  Waiting for authorization...              ║
   ║  ⏱  Code expires in: 15:00                 ║
   ║                                            ║
   ╚════════════════════════════════════════════╝
   ```

6. **In your browser:**
   - Go to http://51.91.97.241/device
   - Enter the code shown in the terminal (e.g., WXYZ-1234)
   - You'll be redirected to Mastodon
   - Login to your Mastodon account (if not already logged in)
   - Authorize terminalpub

7. **Back in SSH terminal:**
   - The polling will detect authorization (checks every 5 seconds)
   - You'll see a success screen:
   ```
   ╔════════════════════════════════════════════╗
   ║        🎉 Successfully Logged In!          ║
   ╠════════════════════════════════════════════╣
   ║                                            ║
   ║  Welcome, @username@mastodon.social        ║
   ║                                            ║
   ║  Your SSH key has been associated with     ║
   ║  your account. Next time you connect,      ║
   ║  you'll be automatically logged in!        ║
   ║                                            ║
   ╚════════════════════════════════════════════╝
   ```

### Test 2: Returning User (Auto-Login)

1. **Disconnect and reconnect via SSH:**
   ```bash
   ssh 51.91.97.241
   ```

2. **You should be automatically logged in!**
   - The system recognizes your SSH public key
   - No need to re-authorize with Mastodon
   - Directly taken to authenticated screen

### Test 3: Anonymous Mode

1. **Connect via SSH**
   ```bash
   ssh 51.91.97.241
   ```

2. **Press `A` for Anonymous**
   - Browse without logging in
   - Limited features

### Test 4: Multiple Devices

1. **From another machine/SSH key:**
   - Connect via SSH
   - Login with same Mastodon account
   - New SSH key will be associated
   - Both devices can now auto-login

## 🔍 What to Check

### Database Verification

Connect to PostgreSQL on VPS:
```bash
ssh -p 2222 ubuntu@51.91.97.241
sudo -u postgres psql terminalpub
```

Check tables:
```sql
-- Check if user was created
SELECT id, username, primary_mastodon_acct, created_at FROM users;

-- Check if SSH key was associated
SELECT user_id, fingerprint, key_type, last_used_at FROM user_ssh_keys;

-- Check if token was stored
SELECT user_id, instance_url, username, is_primary FROM mastodon_tokens;

-- Check device codes (should be authorized=true after login)
SELECT user_code, instance_url, authorized, user_id FROM device_codes;

-- Check sessions
SELECT id, user_id, ip_address, created_at FROM sessions;
```

### Logs

Check server logs:
```bash
ssh -p 2222 ubuntu@51.91.97.241
sudo journalctl -u terminalpub -f
```

Look for:
- Device code generation
- OAuth callback success
- User creation
- SSH key association

## 🐛 Known Issues / Limitations

1. **No Mastodon Instance Validation**
   - The system doesn't pre-validate if an instance exists
   - Will fail at OAuth step if instance is invalid

2. **No Token Encryption**
   - Tokens are stored in plaintext in PostgreSQL
   - TODO: Add encryption at rest (Phase 2 enhancement)

3. **No Multi-Account Support in TUI**
   - Users can only use their primary Mastodon account in SSH
   - Multiple accounts stored but not selectable yet

4. **No Session Expiry UI**
   - Sessions expire after 24h but no warning shown
   - User must re-authenticate

5. **No Error Handling for Network Issues**
   - If Mastodon is down, errors are generic
   - TODO: Better error messages

## ✅ Success Criteria

Phase 2 is complete when:

- [x] User can login via Mastodon OAuth Device Flow
- [x] Device code is generated and displayed
- [x] Web page accepts device code
- [x] OAuth flow redirects to Mastodon
- [x] Callback receives auth code and exchanges for token
- [x] User account is created
- [x] SSH key is associated with user
- [x] Returning users auto-login via SSH key
- [x] Sessions are tracked in Redis + PostgreSQL
- [ ] Manual testing confirms end-to-end flow works
- [ ] At least one successful login from real Mastodon instance

## 🎯 Next Steps (Phase 3)

Once login is verified working:

1. **ActivityPub Integration**
   - WebFinger endpoint
   - Actor endpoints
   - Inbox/Outbox handlers
   - HTTP signatures

2. **Feed Implementation**
   - Fetch home timeline from Mastodon
   - Display in TUI
   - Navigation (up/down, pagination)

3. **Post Creation**
   - Compose screen in TUI
   - Post to Mastodon via API
   - Federated to ActivityPub

## 📝 Test Mastodon Instances

For testing, try these public instances:

- **mastodon.social** - Largest instance (might be slow)
- **fosstodon.org** - FOSS-focused
- **mas.to** - General purpose
- **mastodon.online** - General purpose
- **mstdn.social** - General purpose

## 🔐 Security Notes

- SSH keys are securely bound to accounts
- Only the holder of the private SSH key can access the account
- Mastodon passwords are never exposed to terminalpub
- OAuth tokens are stored server-side only
- Sessions have 24h expiry for security

---

**Status**: Phase 2 Implementation Complete ✅  
**Deployed**: Yes, live at 51.91.97.241  
**Ready for Testing**: Yes
