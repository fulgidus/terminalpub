# Phase 4 - Interactive Features & Content Creation

## Overview
Phase 4 focuses on transforming terminalpub from a **read-only** feed viewer into a **fully interactive** social platform. Users will be able to create posts, reply to conversations, view profiles, and receive notifications.

## Start Date
**November 22, 2025**

## Goals
1. Enable users to create and publish posts to the Fediverse
2. Support threaded conversations with replies
3. Provide user profile viewing
4. Implement conversation thread navigation
5. Add notifications support
6. (Stretch) Search functionality

## Priority Features

### 1. Post Composition Screen (HIGH PRIORITY)
**Goal**: Allow users to write and publish posts to Mastodon/Fediverse

#### Features
- Multi-line text input area
- Real-time character counter (500 char limit for Mastodon)
- Visibility selector (Public, Unlisted, Followers-only, Direct)
- Content warning (CW) toggle
- Cancel/Post actions
- Preview before posting
- Visual feedback on success/error

#### UI Mockup
```
╔══════════════════════════════════════════════════════════╗
║                    Compose New Post                      ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  ┌─────────────────────────────────────────────────────┐ ║
║  │ What's on your mind?                                │ ║
║  │                                                     │ ║
║  │ Just deployed Phase 4 of terminalpub! Now you       │ ║
║  │ can post directly from your terminal. 🚀            │ ║
║  │                                                     │ ║
║  │                                                     │ ║
║  └─────────────────────────────────────────────────────┘ ║
║                                                          ║
║  Characters: 98/500                                      ║
║                                                          ║
║  Visibility: [Public ▼]                                  ║
║  Content Warning: [ ] Add CW                             ║
║                                                          ║
║  [Ctrl+P] Post  [Ctrl+W] Add CW  [Esc] Cancel            ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
```

#### Implementation
- New file: `internal/ui/compose.go`
- Use Bubbletea textarea component
- Call `MastodonService.PostStatus()`
- Validate character count client-side
- Handle API errors gracefully

#### Keyboard Shortcuts
- **Ctrl+P** - Post/Publish
- **Ctrl+W** - Toggle content warning
- **Ctrl+V** - Change visibility
- **Esc** - Cancel and return to menu

---

### 2. Reply to Posts (HIGH PRIORITY)
**Goal**: Enable threaded conversations by replying to existing posts

#### Features
- Reply from feed view (press 'R' on selected post)
- Show parent post context
- Character counter
- Automatic @mention of original author
- Thread detection
- Success confirmation

#### UI Mockup
```
╔══════════════════════════════════════════════════════════╗
║                    Reply to Post                         ║
╠══════════════════════════════════════════════════════════╣
║  Replying to:                                            ║
║  ┌────────────────────────────────────────────────────┐ ║
║  │ Alice Johnson @alice@mastodon.social               │ ║
║  │                                                    │ ║
║  │ Just deployed my new SSH-based social network!    │ ║
║  │ Check it out at terminalpub.com 🚀                │ ║
║  └────────────────────────────────────────────────────┘ ║
║                                                          ║
║  Your reply:                                             ║
║  ┌────────────────────────────────────────────────────┐ ║
║  │ @alice@mastodon.social This looks amazing! I love │ ║
║  │ the terminal-based approach. Can't wait to try it │ ║
║  │                                                    │ ║
║  └────────────────────────────────────────────────────┘ ║
║                                                          ║
║  Characters: 112/500                                     ║
║                                                          ║
║  [Ctrl+P] Reply  [Esc] Cancel                            ║
╚══════════════════════════════════════════════════════════╝
```

#### Implementation
- Extend `internal/ui/compose.go` with reply mode
- Add `replyToID` parameter
- Modify `MastodonService.PostStatus()` to accept `in_reply_to_id`
- Show parent post as context
- Auto-populate @mentions

#### Feed Integration
- Add 'R' key handler in feed view
- Pass selected post to compose screen
- Return to feed after successful reply

---

### 3. User Profile Viewing (MEDIUM PRIORITY)
**Goal**: View user profiles and their recent posts

#### Features
- Display user avatar (ASCII art or placeholder)
- Show bio, follower count, following count
- Display pinned posts
- Show recent posts (20 most recent)
- Follow/Unfollow button (if not current user)
- Navigate to profile from feed (press 'P' on post)

#### UI Mockup
```
╔══════════════════════════════════════════════════════════╗
║                    User Profile                          ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  Alice Johnson                                           ║
║  @alice@mastodon.social                                  ║
║                                                          ║
║  Software engineer | Terminal enthusiast | Fediverse    ║
║  advocate                                                ║
║                                                          ║
║  🔗 https://alicejohnson.dev                            ║
║  📍 San Francisco, CA                                    ║
║                                                          ║
║  Following: 342   Followers: 1,204   Posts: 3,892       ║
║                                                          ║
║  [Following ✓]                                           ║
║                                                          ║
║  ━━━━━━━━━━━━━━━━ Recent Posts ━━━━━━━━━━━━━━━━         ║
║                                                          ║
║  Just deployed my new SSH-based social network! ...     ║
║  ❤ 42    🔄 15    💬 8                                   ║
║  ────────────────────────────────────────────────────   ║
║  Working on Phase 4 features today. Excited to ...      ║
║  ❤ 28    🔄 6     💬 3                                   ║
║                                                          ║
║  ↑/↓ Scroll  [F] Follow/Unfollow  [B] Back              ║
╚══════════════════════════════════════════════════════════╝
```

#### Implementation
- New file: `internal/ui/profile.go`
- Add `MastodonService.GetAccount(accountID)`
- Add `MastodonService.GetAccountStatuses(accountID)`
- Add `MastodonService.FollowAccount(accountID)`
- Add `MastodonService.UnfollowAccount(accountID)`
- Extract account ID from post selection

---

### 4. Conversation Thread Viewing (MEDIUM PRIORITY)
**Goal**: View full conversation threads in context

#### Features
- Show parent post (if reply)
- Show all replies in thread
- Hierarchical indentation
- Navigate between posts in thread
- Reply from thread view

#### UI Mockup
```
╔══════════════════════════════════════════════════════════╗
║                    Conversation Thread                   ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  Alice Johnson @alice@mastodon.social                    ║
║  Just deployed my new SSH-based social network! ...     ║
║  ❤ 42    🔄 15    💬 8                                   ║
║                                                          ║
║    ┗━▶ Bob Williams @bob@fosstodon.org                  ║
║       This looks amazing! How does federation work?     ║
║       ❤ 12    🔄 2     💬 1                              ║
║                                                          ║
║         ┗━▶ Alice Johnson @alice@mastodon.social        ║
║            It uses ActivityPub! Messages are sent ...   ║
║            ❤ 8     🔄 1     💬 0                         ║
║                                                          ║
║    ┗━▶ Carol Davis @carol@pixelfed.social               ║
║       Can I post photos from the terminal?              ║
║       ❤ 5     🔄 0     💬  2                              ║
║                                                          ║
║  ↑/↓ Navigate  [R] Reply  [B] Back                      ║
╚══════════════════════════════════════════════════════════╝
```

#### Implementation
- New file: `internal/ui/thread.go`
- Add `MastodonService.GetStatusContext(statusID)`
- Build tree structure from ancestors/descendants
- Render with indentation
- Handle deep nesting (>5 levels)

#### Feed Integration
- Add 'T' key handler in feed view
- Pass selected post ID
- Return to feed after viewing thread

---

### 5. Notifications (MEDIUM PRIORITY)
**Goal**: View mentions, likes, boosts, and follows

#### Features
- Notification types: mention, favourite, reblog, follow
- Unread count indicator
- Mark as read
- Navigate to related post/profile
- Real-time updates (polling every 30s)

#### UI Mockup
```
╔══════════════════════════════════════════════════════════╗
║                  Notifications (5 new)                   ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║ ► ❤ Bob Williams favourited your post                   ║
║   "Just deployed my new SSH-based social network! ..."  ║
║   2 minutes ago                                          ║
║   ────────────────────────────────────────────────────  ║
║                                                          ║
║   🔄 Carol Davis boosted your post                       ║
║   "Just deployed my new SSH-based social network! ..."  ║
║   5 minutes ago                                          ║
║   ────────────────────────────────────────────────────  ║
║                                                          ║
║   💬 Dave Wilson mentioned you                           ║
║   "@alice This is so cool! Can I contribute?"           ║
║   10 minutes ago                                         ║
║   ────────────────────────────────────────────────────  ║
║                                                          ║
║   👤 Eve Martinez started following you                  ║
║   15 minutes ago                                         ║
║   ────────────────────────────────────────────────────  ║
║                                                          ║
║  ↑/↓ Navigate  [Enter] View  [M] Mark read  [B] Back   ║
╚══════════════════════════════════════════════════════════╝
```

#### Implementation
- New file: `internal/ui/notifications.go`
- Add `MastodonService.GetNotifications()`
- Add `MastodonService.DismissNotification(notifID)`
- Add notification badge to main menu
- Implement polling mechanism

---

### 6. Search (LOW PRIORITY / STRETCH GOAL)
**Goal**: Search for hashtags, users, and posts

#### Features
- Search input field
- Results categorized by type (accounts, hashtags, statuses)
- Navigate to profiles/posts from results
- Search history

#### UI Mockup
```
╔══════════════════════════════════════════════════════════╗
║                         Search                           ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  Query: #activitypub                                     ║
║  ┌────────────────────────────────────────────────────┐ ║
║  │ #activitypub                                       │ ║
║  └────────────────────────────────────────────────────┘ ║
║                                                          ║
║  Results:                                                ║
║                                                          ║
║  Hashtags                                                ║
║  ────────────────────────────────────────────────────   ║
║  #activitypub - 1,234 posts                              ║
║  #ActivityPubDev - 456 posts                             ║
║                                                          ║
║  Accounts                                                ║
║  ────────────────────────────────────────────────────   ║
║  @activitypub@mastodon.social                            ║
║  Official ActivityPub account                            ║
║                                                          ║
║  Posts                                                   ║
║  ────────────────────────────────────────────────────   ║
║  Alice Johnson: "Learning about #activitypub ..."       ║
║                                                          ║
║  ↑/↓ Navigate  [Enter] Open  [Esc] Back                 ║
╚══════════════════════════════════════════════════════════╝
```

#### Implementation
- New file: `internal/ui/search.go`
- Add `MastodonService.Search(query, type)`
- Support search types: accounts, hashtags, statuses
- Implement result navigation

---

## Technical Architecture

### New Files to Create
1. `internal/ui/compose.go` - Post composition screen
2. `internal/ui/profile.go` - User profile viewer
3. `internal/ui/thread.go` - Conversation thread viewer
4. `internal/ui/notifications.go` - Notifications screen
5. `internal/ui/search.go` - Search interface (stretch)

### Extensions to Existing Files
1. `internal/services/mastodon.go` - Add new API methods:
   - `PostStatus(content, visibility, inReplyToID, contentWarning)`
   - `GetAccount(accountID)`
   - `GetAccountStatuses(accountID)`
   - `FollowAccount(accountID)`
   - `UnfollowAccount(accountID)`
   - `GetStatusContext(statusID)`
   - `GetNotifications()`
   - `DismissNotification(notifID)`
   - `Search(query, searchType)`

2. `internal/ui/tui.go` - Add navigation to new screens:
   - Compose screen (P key from main menu)
   - Reply screen (R key from feed)
   - Profile screen (P key from feed post)
   - Thread screen (T key from feed post)
   - Notifications screen (N key from main menu)
   - Search screen (/ key from main menu)

3. `internal/ui/feed.go` - Add new keyboard shortcuts:
   - R - Reply to selected post
   - P - View post author's profile
   - T - View conversation thread

### UI Navigation Flow
```
Main Menu
  ├── [F] Feed (existing)
  │     ├── [R] Reply to post → Compose Screen (reply mode)
  │     ├── [P] View profile → Profile Screen
  │     └── [T] View thread → Thread Screen
  ├── [P] Compose new post → Compose Screen
  ├── [N] Notifications → Notifications Screen
  ├── [/] Search → Search Screen (stretch)
  └── [Q] Quit
```

### State Management
- Add `composing`, `viewingProfile`, `viewingThread`, `viewingNotifications`, `searching` states to TUI model
- Pass context between screens (e.g., post ID for replies, account ID for profiles)
- Handle back navigation consistently

---

## Testing Strategy

### Unit Tests
- `TestComposeScreen_CharacterCount()`
- `TestComposeScreen_VisibilityToggle()`
- `TestMastodonService_PostStatus()`
- `TestMastodonService_GetStatusContext()`
- `TestProfileScreen_FollowUnfollow()`

### Integration Tests
- Post creation end-to-end
- Reply threading
- Profile viewing with real API
- Notification fetching

### Manual Testing Checklist
- [ ] Compose and post a new status
- [ ] Reply to an existing post
- [ ] View user profile
- [ ] Follow/unfollow user
- [ ] View conversation thread
- [ ] View notifications
- [ ] Mark notifications as read
- [ ] Search for hashtags (if implemented)
- [ ] Search for users (if implemented)
- [ ] Test with various character counts (1, 250, 500, 501)
- [ ] Test with special characters and emoji
- [ ] Test content warnings
- [ ] Test visibility options
- [ ] Test error handling (network failures, API errors)
- [ ] Test on different terminal sizes

---

## Error Handling

### Common Scenarios
1. **API Errors** - Display user-friendly messages
2. **Network Timeouts** - Show retry option
3. **Character Limit Exceeded** - Prevent posting, show error
4. **Invalid Account ID** - Show error, return to previous screen
5. **Rate Limiting** - Display wait time

### User Feedback
- Status messages in footer
- Loading indicators for API calls
- Success confirmations
- Error messages with actionable steps

---

## Performance Considerations

### Optimization Strategies
1. **Caching** - Cache user profiles for 5 minutes
2. **Lazy Loading** - Load thread replies on demand
3. **Debouncing** - Character counter updates
4. **Background Polling** - Notifications check every 30s (when on menu)
5. **Pagination** - Thread replies if >50

### Resource Constraints
- Memory: Keep only necessary data in memory
- Network: Minimize API calls
- Rendering: Optimize view updates

---

## Documentation Updates

### Files to Update
1. `README.md` - Add Phase 4 features to feature list
2. `DEPLOYMENT_PHASE4.md` - Deployment guide for Phase 4
3. `AGENTS.md` - Update with new commands and patterns
4. `PHASE4_COMPLETE.md` - Completion summary (at end)

### User Documentation
- Update keyboard shortcuts reference
- Add screenshots/examples
- Document visibility options
- Explain threading behavior

---

## Success Metrics

Phase 4 will be considered complete when:
- ✅ Users can compose and post new statuses
- ✅ Users can reply to existing posts
- ✅ Users can view user profiles
- ✅ Users can view conversation threads
- ✅ Users can view notifications
- ✅ All features work without crashes
- ✅ Error handling is comprehensive
- ✅ Documentation is updated
- ✅ Code is tested and reviewed
- ✅ Deployed to VPS successfully

---

## Timeline Estimate

### Week 1 (Nov 22-28)
- Day 1-2: Compose screen implementation
- Day 3-4: Reply functionality
- Day 5: Testing and refinement

### Week 2 (Nov 29-Dec 5)
- Day 1-2: User profile viewer
- Day 3: Conversation thread viewer
- Day 4-5: Notifications

### Week 3 (Dec 6-12)
- Day 1-2: Search (if time permits)
- Day 3-4: Testing, bug fixes
- Day 5: Documentation, deployment

**Total Estimated Time**: 2-3 weeks

---

## Risks & Mitigation

### Potential Risks
1. **Mastodon API Complexity** - Some endpoints may require special handling
   - Mitigation: Test with multiple instances, read API docs thoroughly
   
2. **UI Complexity** - Multi-line input in Bubbletea can be tricky
   - Mitigation: Use existing textarea components, research examples

3. **Thread Rendering** - Deep thread nesting can be challenging
   - Mitigation: Limit depth to 5 levels, implement collapse/expand

4. **Performance** - Too many API calls could slow down UI
   - Mitigation: Implement caching, debouncing, lazy loading

---

## Next Steps

1. **START**: Implement compose screen (`internal/ui/compose.go`)
2. Add `PostStatus()` method to `internal/services/mastodon.go`
3. Integrate compose screen into TUI navigation
4. Test post creation with real Mastodon account
5. Move to reply functionality

Let's build an amazing interactive experience! 🚀
