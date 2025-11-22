# Phase 3 Deployment Guide

## Build Information
- **Version**: Phase 3 - Complete
- **Build Date**: 2025-11-22
- **Binary Size**: ~30MB
- **Git Commit**: fd1c145

## New Features in This Release
1. ✅ Mastodon timeline feed viewer
2. ✅ Three timeline types (Home, Local, Federated)
3. ✅ Post navigation with keyboard
4. ✅ Like/favourite posts
5. ✅ Boost/reblog posts
6. ✅ Pagination with load more
7. ✅ Real-time status feedback

## Deployment Steps

### 1. Build Binary (Local)
```bash
cd /home/fulgidus/Documents/terminalpub
go build -o terminalpub ./cmd/server
```

### 2. Transfer to VPS
```bash
# Option A: SCP
scp terminalpub root@51.91.97.241:/tmp/terminalpub-new

# Option B: Rsync
rsync -avz terminalpub root@51.91.97.241:/tmp/terminalpub-new
```

### 3. Deploy on VPS
```bash
# SSH into VPS
ssh root@51.91.97.241

# Stop existing service
systemctl stop terminalpub

# Backup old binary
cp /opt/terminalpub/bin/terminalpub /opt/terminalpub/bin/terminalpub.backup

# Install new binary
mv /tmp/terminalpub-new /opt/terminalpub/bin/terminalpub
chmod +x /opt/terminalpub/bin/terminalpub
chown terminalpub:terminalpub /opt/terminalpub/bin/terminalpub

# Start service
systemctl start terminalpub

# Check status
systemctl status terminalpub
journalctl -u terminalpub -f
```

### 4. Verify Deployment
```bash
# Test SSH connection
ssh 51.91.97.241

# Expected behavior:
# 1. Welcome screen appears
# 2. Press L to login
# 3. Enter Mastodon instance
# 4. Complete OAuth flow
# 5. Press F to view feed
# 6. Navigate with arrow keys
# 7. Press X to like, S to boost
# 8. Press M to load more posts
```

## Testing Checklist

### Feed Functionality
- [ ] Feed screen loads from authenticated menu (F key)
- [ ] Home timeline displays correctly
- [ ] Local timeline switches properly (L key)
- [ ] Federated timeline switches properly (F key)
- [ ] Navigation works (↑/↓ keys)
- [ ] Post selection indicator visible (►)
- [ ] HTML content is stripped properly
- [ ] Word wrapping works correctly

### Interactions
- [ ] Like post shows confirmation (X key)
- [ ] Boost post shows confirmation (S key)
- [ ] Status message displays in footer
- [ ] Error messages display properly

### Pagination
- [ ] Load more button appears ([M] hint)
- [ ] Loading more shows progress message
- [ ] New posts append to feed
- [ ] "No more posts" message when done
- [ ] Scroll position maintained during load

### Edge Cases
- [ ] Empty timeline handled gracefully
- [ ] Network errors display properly
- [ ] API rate limits handled
- [ ] Large posts truncate correctly
- [ ] Boost detection works (shows "🔄 X boosted")

## Rollback Procedure

If issues occur:
```bash
# SSH into VPS
ssh root@51.91.97.241

# Stop service
systemctl stop terminalpub

# Restore backup
cp /opt/terminalpub/bin/terminalpub.backup /opt/terminalpub/bin/terminalpub

# Start service
systemctl start terminalpub
```

## Configuration

No configuration changes required for Phase 3.

Existing `config/config.yaml` settings are sufficient.

## Database

No new migrations required for Phase 3.

All necessary tables already exist from Phase 2.

## Known Issues

None identified in Phase 3 implementation.

## Performance Notes

- Timeline fetches are async (non-blocking UI)
- Default: 20 posts per load
- Pagination uses Mastodon's maxID for efficiency
- No caching yet (future enhancement)

## Next Steps (Phase 4)

- [ ] Post composition screen
- [ ] Reply to posts
- [ ] User profile viewing
- [ ] Thread viewing
- [ ] Notifications
- [ ] Search functionality

## Support

If issues occur:
1. Check logs: `journalctl -u terminalpub -f`
2. Check database connectivity
3. Verify Mastodon API access
4. Check network/firewall rules

## Success Metrics

Phase 3 is successful when:
- ✅ Users can view their Mastodon home feed
- ✅ Users can switch between timeline types
- ✅ Users can like and boost posts
- ✅ Pagination works smoothly
- ✅ No crashes or errors during normal use
