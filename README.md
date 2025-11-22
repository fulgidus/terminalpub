# terminalpub

> ActivityPub for your terminal

[![License](https://img.shields.io/badge/license-AGPLv3-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev)

A terminal-first federated social network powered by SSH and ActivityPub. Connect to the fediverse without leaving your shell.

   ```bash
   ssh terminalpub.com
   ```

## Features

- 🔐 **Mastodon Login** - OAuth Device Flow for secure authentication
- 🌐 **Full ActivityPub** - Native federation with Mastodon, Pleroma, and the entire fediverse
- 💬 **Chat Roulette** - Anonymous random conversations via SSH
- 📝 **Post & Share** - Create posts visible across the fediverse
- #️⃣ **Hashtags** - Full hashtag support with mouse-clickable tags
- 🔄 **Unified Feed** - See posts from your Mastodon following
- 👤 **Anonymous Mode** - Browse without login
- ⬆️ **Upvotes & Comments** - Engage with federated content
- 🎨 **Beautiful TUI** - Crafted with Charm libraries

## Quick Start

```bash
# Clone repository
git clone https://github.com/fulgidus/terminalpub
cd terminalpub

# Copy config
cp config/config.example.yaml config/config.yaml

# Start dependencies (PostgreSQL + Redis)
docker-compose up -d

# Run database migrations
make migrate-up

# Run server
make run
```

Connect via SSH:
```bash
ssh localhost
```

## SSH Key Requirement

**terminalpub requires SSH public key authentication for all connections.**

If you don't have an SSH key pair, generate one:

```bash
# Generate an ED25519 key (recommended)
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519

# Or generate an RSA key (alternative)
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa
```

Press Enter when prompted for a passphrase (or set one for extra security).

Your SSH key will be automatically associated with your account after your first Mastodon login. On subsequent connections, you'll be automatically logged in!

## User Experience

### First Time Connection

```
$ ssh terminalpub.com

╔════════════════════════════════╗
║      Welcome to terminalpub!   ║
╠════════════════════════════════╣
║  [L] Login with Mastodon       ║
║  [A] Continue anonymously      ║
║  [Q] Quit                      ║
╚════════════════════════════════╝

> L

Mastodon instance: mastodon.social

╔══════════════════════════════════════════╗
║         Login to Mastodon                ║
╠══════════════════════════════════════════╣
║  1. Visit: https://terminalpub.com/device║
║                                          ║
║  2. Enter code: WXYZ-1234                ║
║                                          ║
║  3. Authorize terminalpub                ║
║                                          ║
║  Waiting for authorization...            ║
╚══════════════════════════════════════════╝

✓ Logged in as @alice@mastodon.social

╔════════════════════════════════╗
║          Main Menu             ║
╠════════════════════════════════╣
║  [P] Compose new post          ║
║  [F] View your Mastodon feed   ║
║  [N] View notifications        ║
║  [X] Logout                    ║
║  [Q] Quit                      ║
╚════════════════════════════════╝
```

### Feed Navigation (Phase 3)

Once authenticated, press **[F]** to view your Mastodon feed:

```
╔════════════════════════════════════════════╗
║          Home Timeline (20 posts)          ║
╠════════════════════════════════════════════╣
║                                            ║
║ ► Alice Johnson                            ║
║   @alice@mastodon.social                   ║
║                                            ║
║   Just deployed my new SSH-based social    ║
║   network! Check it out at terminalpub.com ║
║                                            ║
║   ❤ 42    🔄 15    💬 8                    ║
║────────────────────────────────────────────║
║                                            ║
║   Bob Williams                             ║
║   @bob@fosstodon.org                       ║
║                                            ║
║   Terminal UIs are making a comeback!      ║
║   Love the retro aesthetic 🎨              ║
║                                            ║
║   ❤ 128   🔄 34    💬 22                   ║
║────────────────────────────────────────────║
║                                            ║
║  ↑/↓ Navigate  [H]ome [L]ocal [F]ederated ║
║  [X] Like  [S] Boost  [R] Refresh          ║
║  Post 1/20  [B]ack  [Q]uit                 ║
║                                            ║
║  Status: Ready                             ║
╚════════════════════════════════════════════╝
```

**Feed Controls:**
- **↑/↓ or K/J** - Navigate between posts
- **H** - Switch to Home timeline (following only)
- **L** - Switch to Local timeline (instance posts)
- **F** - Switch to Federated timeline (all public posts)
- **T** - View full conversation thread
- **P** - View user profile
- **R** - Reply to the selected post
- **X** - Like/favourite the selected post
- **S** - Boost/reblog the selected post
- **Ctrl+R** - Refresh feed
- **B** - Back to main menu
- **Q** - Quit

The feed shows 5 posts at a time with automatic scrolling. Posts display:
- Author name and handle
- Post content (word-wrapped)
- Interaction counts (likes, boosts, replies)
- Selection indicator (►) for the current post
- [*] marker on likes/boosts you've already interacted with

**Infinite Scrolling:** The feed automatically loads more posts when you scroll near the end.

### Post Composition (Phase 4)

From the main menu, press **[P]** to compose a new post:

```
╔══════════════════════════════════════════════════════════╗
║                    Compose New Post                      ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  Write your post:                                        ║
║  ┌────────────────────────────────────────────────────┐ ║
║  │ What's on your mind?                               │ ║
║  │                                                    │ ║
║  │                                                    │ ║
║  └────────────────────────────────────────────────────┘ ║
║                                                          ║
║  Characters: 0/500                                       ║
║                                                          ║
║  Visibility: [public ▼]                                  ║
║  Content Warning: [ ] Add CW                             ║
║                                                          ║
║  [Ctrl+P] Post  [Ctrl+W] Toggle CW  [Ctrl+V] Visibility ║
║  [Esc] Cancel                                            ║
╚══════════════════════════════════════════════════════════╝
```

**Compose Controls:**
- **Type** - Enter your post content (up to 500 characters)
- **Ctrl+P** - Publish the post
- **Ctrl+W** - Toggle content warning
- **Ctrl+V** - Cycle visibility (public → unlisted → private → direct)
- **Esc** - Cancel and return to menu

**Reply to Posts:**
From the feed, press **[R]** on any post to reply. The compose screen will show the original post context and automatically mention the author.

### Conversation Threads (Phase 4)

Press **[T]** on any post in the feed to view the full conversation thread:

```
                    Conversation Thread

  Alice Johnson @alice@mastodon.social [Original Post]
  Just deployed my new SSH-based social network!
  Likes: 42  Boosts: 15  Replies: 8

    └─▶ Bob Williams @bob@fosstodon.org
        This looks amazing! How does federation work?
        Likes: 12  Boosts: 2  Replies: 1

      └─▶ Alice Johnson @alice@mastodon.social
          It uses ActivityPub! Messages are sent...
          Likes: 8  Boosts: 1  Replies: 0

    └─▶ Carol Davis @carol@pixelfed.social
        Can I post photos from the terminal?
        Likes: 5  Boosts: 0  Replies: 2

  ↑/↓ Navigate  [R] Reply  [ESC] Back  [O] View in Browser
```

**Thread Controls:**
- **↑/↓ or K/J** - Navigate between posts in the thread
- **R** - Reply to the selected post
- **O** - Show URL to view in browser
- **Esc** - Return to feed

The thread view shows:
- Hierarchical structure with indentation
- Parent posts (ancestors)
- The original/root post marked with [Original Post]
- All replies (descendants) in tree order
- Full conversation context

### User Profiles (Phase 4)

Press **[P]** on any post in the feed to view the author's profile:

```
                    User Profile

  Alice Johnson
  @alice@mastodon.social

  Software engineer | Terminal enthusiast | Fediverse advocate

  Following: 342   Followers: 1,204   Posts: 3,892

  [Following ✓]

  ────────────────────────────────────────
                Recent Posts
  ────────────────────────────────────────

  ► Just deployed my new SSH-based social network!
    Likes: 42  Boosts: 15  Replies: 8
    ────────────────────────────
    Working on Phase 4 features today. Excited to...
    Likes: 28  Boosts: 6  Replies: 3

  ↑/↓ Navigate  [F] Follow/Unfollow  [R] Reply  [T] Thread  [ESC] Back
```

**Profile Controls:**
- **↑/↓ or K/J** - Navigate between posts
- **F** - Follow or unfollow the user
- **R** - Reply to the selected post
- **T** - View thread for the selected post
- **Esc or B** - Return to feed

The profile view shows:
- User's display name and handle
- Bio/description
- Follower and following counts
- Follow status and button
- Recent posts (20 most recent)
- Navigate and interact with posts directly from profile

### Notifications (Phase 4)

Press **[N]** from the main menu to view your notifications:

```
                  Notifications (12)

  ► Like: Bob Williams liked your post
      "Just deployed my new SSH-based social network! ..."
      2 minutes ago
      ────────────────────────────

    Boost: Carol Davis boosted your post
      "Just deployed my new SSH-based social network! ..."
      5 minutes ago
      ────────────────────────────

    Reply: Dave Wilson mentioned you
      "@alice This is so cool! Can I contribute?"
      10 minutes ago
      ────────────────────────────

    Follow: Eve Martinez started following you
      15 minutes ago
      ────────────────────────────

  ↑/↓ Navigate  [Enter] View  [D] Dismiss  [C] Clear All  [ESC] Back
```

**Notification Controls:**
- **↑/↓ or K/J** - Navigate between notifications
- **Enter** - View the related post (thread) or profile (for follows)
- **D** - Dismiss the selected notification
- **C** - Clear all notifications
- **Ctrl+R** - Refresh notifications
- **Esc or B** - Return to main menu

Notification types displayed:
- **Mentions/Replies** - Someone mentioned you or replied to your post
- **Likes** - Someone liked your post
- **Boosts** - Someone boosted/reblogged your post
- **Follows** - Someone started following you
- **Follow Requests** - Someone requested to follow you (private accounts)
- **Poll Results** - A poll you voted in has ended

Features:
- Shows relative timestamps (e.g., "5 minutes ago")
- Auto-loads more notifications when scrolling near the end
- Press Enter on mentions to view the conversation thread
- Press Enter on follows to view the follower's profile

## Architecture

```
┌─────────────┐                  ┌──────────────┐
│  SSH Client │◄────────────────►│ terminalpub  │
└─────────────┘                  │  SSH Server  │
                                 └──────┬───────┘
                                        │
                    ┌───────────────────┼──────────────────┐
                    │                   │                  │
             ┌──────▼──────┐      ┌─────▼──────┐     ┌─────▼──────┐
             │  PostgreSQL │      │   Redis    │     │ActivityPub │
             │   Database  │      │   Cache    │     │ Federation │
             └─────────────┘      └────────────┘     └────────────┘
                                                           │
                                                    ┌──────▼──────┐
                                                    │ Mastodon    │
                                                    │ Pleroma     │
                                                    │ Pixelfed    │
                                                    │ Fediverse   │
                                                    └─────────────┘
```

### Component Overview

- **SSH Server** (Wish) - Handles terminal connections and TUI rendering
- **HTTP Server** - Serves ActivityPub endpoints and OAuth web pages
- **PostgreSQL** - Stores users, posts, follows, activities
- **Redis** - Caching, sessions, real-time features (chatroulette queue)
- **Background Workers** - Process ActivityPub inbox/outbox, delivery queue

## Tech Stack

- **Go 1.21+** - Primary language
- **Charm Libraries**
  - [Wish](https://github.com/charmbracelet/wish) - SSH server
  - [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
  - [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- **PostgreSQL 15+** - Relational database
- **Redis 7+** - Cache and real-time data
- **ActivityPub** - W3C federation protocol

## Project Structure

```
terminalpub/
├── cmd/
│   ├── server/          # Main SSH+HTTP server
│   ├── worker/          # Background federation worker
│   └── migrate/         # Database migration tool
├── internal/
│   ├── activitypub/     # ActivityPub protocol implementation
│   ├── auth/            # Authentication & OAuth Device Flow
│   ├── db/              # Database layer (PostgreSQL + Redis)
│   ├── handlers/        # SSH & HTTP request handlers
│   ├── models/          # Data models
│   ├── services/        # Business logic
│   ├── ui/              # TUI components (Bubbletea)
│   └── workers/         # Background job workers
├── migrations/          # SQL database migrations
├── config/              # Configuration files
├── web/                 # HTML templates for OAuth flow
└── docs/                # Documentation
```

## Configuration

See `config/config.example.yaml` for all available options.

Key configuration areas:
- **Server** - Domain, ports (SSH: 2222, HTTP: 443)
- **Database** - PostgreSQL and Redis connection strings
- **OAuth** - Device flow settings, callback URLs
- **ActivityPub** - Federation settings, user agent, workers
- **Features** - Enable/disable chatroulette, anonymous posting
- **Security** - Rate limiting, blocked instances

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose (for local dev)

### Running Locally

```bash
# Install dependencies
make install-deps

# Start PostgreSQL & Redis
make docker-up

# Run migrations
make migrate-up

# Run server in development mode
make dev

   # In another terminal, connect via SSH
   ssh localhost
   ```

### Available Make Commands

```bash
make help           # Show all available commands
make build          # Build binary
make run            # Run server
make dev            # Run with auto-reload (air)
make test           # Run tests
make migrate-up     # Run database migrations
make migrate-down   # Rollback migrations
make docker-up      # Start Docker services
make docker-down    # Stop Docker services
make lint           # Run linter
make format         # Format code
```

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md) - System design and components
- [Deployment Guide](docs/DEPLOYMENT.md) - Production deployment instructions
- [ActivityPub Implementation](docs/ACTIVITYPUB.md) - Federation details
- [API Reference](docs/API.md) - HTTP API documentation
- [Contributing Guide](docs/CONTRIBUTING.md) - How to contribute

## Roadmap

### Phase 1: Foundation (Weeks 1-2)
- [x] Project architecture
- [x] Database schema design
- [x] OAuth Device Flow design
- [ ] Core project structure
- [ ] Basic SSH server
- [ ] Database layer (PostgreSQL + Redis)

### Phase 2: Authentication (Weeks 3-4)
- [ ] OAuth Device Flow implementation
- [ ] Mastodon instance app registration
- [ ] Token management and refresh
- [ ] Session handling

### Phase 3: ActivityPub (Weeks 5-6)
- [ ] WebFinger endpoint
- [ ] Actor endpoints
- [ ] Inbox/Outbox handlers
- [ ] HTTP signatures
- [ ] Basic federation

### Phase 4: Core Features (Weeks 7-8)
- [ ] Feed implementation
- [ ] Post creation and display
- [ ] Follow/Unfollow
- [ ] Upvotes (Like activities)
- [ ] Comments (Reply activities)

### Phase 5: Social Features (Weeks 9-10)
- [ ] Chat Roulette
- [ ] Anonymous posting
- [ ] Hashtag parsing and linking
- [ ] Search functionality
- [ ] Notifications

### Phase 6: Federation Workers (Weeks 11-12)
- [ ] Inbox processor
- [ ] Delivery worker with retry logic
- [ ] Sync worker for Mastodon imports
- [ ] Import following/followers

### Phase 7: Polish & Deploy (Weeks 13-14)
- [ ] Error handling and edge cases
- [ ] Rate limiting
- [ ] Moderation tools
- [ ] Performance optimization
- [ ] Production deployment
- [ ] Monitoring and logging

## Security Considerations

- **OAuth Device Flow** - No password sharing, standard OAuth 2.0
- **HTTP Signatures** - All ActivityPub activities are cryptographically signed
- **Rate Limiting** - Per-IP and per-user rate limits
- **Input Sanitization** - All user input is sanitized
- **SQL Injection Protection** - Prepared statements throughout
- **Session Security** - Secure session tokens with expiry
- **Instance Blocking** - Ability to block problematic federated instances

## Performance

Target specifications:
- **Concurrent SSH connections**: 1000+
- **ActivityPub activities/sec**: 100+
- **Average response time**: <100ms
- **Database queries**: Optimized with indexes
- **Caching**: Redis for hot data

## License

AGPLv3 - See [LICENSE](LICENSE)

This project is licensed under the GNU Affero General Public License v3.0. This means:
- ✅ You can use, modify, and distribute this software
- ✅ You can run it for commercial purposes
- ⚠️ If you modify and run it as a network service, you must share your modifications
- ⚠️ All derivatives must also be AGPLv3

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](docs/CONTRIBUTING.md) first.

### How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code of Conduct

This project follows a standard Code of Conduct. Be respectful, inclusive, and professional.

## Community

- **Website**: https://terminalpub.com
- **Repository**: https://github.com/fulgidus/terminalpub
- **Issues**: https://github.com/fulgidus/terminalpub/issues
- **Discussions**: https://github.com/fulgidus/terminalpub/discussions

## Acknowledgments

Built with amazing open source tools:
- [Charm](https://charm.sh) - Beautiful TUI libraries
- [ActivityPub](https://activitypub.rocks) - W3C federation standard
- The Fediverse community for inspiration
- All contributors who help make this project better

## Author

Created by [@fulgidus](https://github.com/fulgidus)

Inspired by the need for a terminal-native way to interact with the fediverse. Because sometimes the best social network is one you can access from `ssh`.

---

**Status**: ✅ SSH server deployed and running!

**Connect**: `ssh 51.91.97.241` ✅ LIVE NOW!

**Requirements**: SSH key pair required. Generate with `ssh-keygen -t ed25519` if you don't have one.
