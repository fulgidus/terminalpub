# terminalpub - Project Specifications

> Comprehensive technical specifications for AI agents and developers

## Project Overview

**Project Name**: terminalpub  
**Domain**: terminalpub.com  
**Repository**: github.com/fulgidus/terminalpub  
**Tagline**: ActivityPub for your terminal  
**Description**: Terminal-first federated social network accessible via SSH with full ActivityPub support

## Core Concept

A social network that you access via SSH (`ssh terminalpub.com`) instead of a web browser. Users can:
1. Login with their existing Mastodon credentials (OAuth Device Flow)
2. See their federated feed from the terminal
3. Post, comment, upvote, and interact with the fediverse
4. Use anonymous features like chat roulette
5. Click hashtags with mouse support in terminal

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **SSH Server**: github.com/charmbracelet/wish
- **TUI Framework**: github.com/charmbracelet/bubbletea
- **Styling**: github.com/charmbracelet/lipgloss
- **Database**: PostgreSQL 15+
- **Cache**: Redis 7+
- **Protocol**: ActivityPub (W3C standard)

### Infrastructure
- **Deployment**: VPS (Hetzner recommended, ~€7/month)
- **OS**: Ubuntu 22.04 or Debian 12
- **Containerization**: Docker & Docker Compose
- **Process Management**: systemd

## Architecture

### High-Level System Architecture

```
┌─────────────────────────────────────────────┐
│           SSH Server (port 2222)            │
│        HTTP Server (port 443)               │
│        (Wish + Chi/Gin)                     │
└──────────────────┬──────────────────────────┘
                   │
        ┌──────────▼──────────┐
        │  Auth Handler       │
        │  - OAuth Device Flow│
        │  - Session Mgmt     │
        └──────────┬──────────┘
                   │
        ┌──────────▼────────────────────┐
        │   Router/Controller           │
        │   - Anonymous mode            │
        │   - Authenticated mode        │
        └──────────┬────────────────────┘
                   │
    ┏━━━━━━━━━━━━━┻━━━━━━━━━━━━━┓
    ┃                            ┃
┌───▼────────┐          ┌────────▼─────┐
│ Anonymous  │          │ Authenticated│
│ Features   │          │ Features     │
│            │          │              │
│-Chatroulette│         │- Feed        │
│-Anon Post  │          │- Post        │
│            │          │- Follow      │
│            │          │- Upvote      │
│            │          │- Comment     │
│            │          │- + Anonymous │
└────┬───────┘          └──────┬───────┘
     │                         │
     └──────────┬──────────────┘
                │
     ┌──────────▼──────────┐
     │   Business Logic    │
     │   - Post service    │
     │   - User service    │
     │   - Follow service  │
     │   - Chat service    │
     │   - Federation svc  │
     │   - Hashtag parser  │
     └──────────┬──────────┘
                │
     ┌──────────▼──────────┐
     │   Data Layer        │
     │   - PostgreSQL      │
     │   - Redis cache     │
     └─────────────────────┘
                │
     ┌──────────▼──────────┐
     │  Background Workers │
     │  - Inbox processor  │
     │  - Delivery worker  │
     │  - Sync worker      │
     └─────────────────────┘
```

### Component Details

#### 1. SSH Server (Wish)
- Handles SSH connections on port 2222
- Renders TUI using Bubbletea
- Manages user sessions
- Supports mouse events for clickable hashtags

#### 2. HTTP Server
- Serves ActivityPub endpoints (port 443)
- OAuth Device Flow web pages
- WebFinger endpoint
- Actor/Inbox/Outbox endpoints

#### 3. Authentication System
- OAuth Device Flow for Mastodon login
- No password storage for Mastodon accounts
- Session management via Redis
- Support for multiple linked Mastodon accounts

#### 4. ActivityPub Federation
- Full W3C ActivityPub implementation
- HTTP signatures for authenticity
- Inbox processing (receive activities)
- Outbox delivery (send activities)
- Support for: Follow, Like, Create, Update, Delete, Announce

#### 5. Background Workers
- Inbox processor: Handles incoming ActivityPub activities
- Delivery worker: Sends activities to remote instances with retry logic
- Sync worker: Imports following/followers from Mastodon

## Database Schema

### Core Tables

#### users
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255),  -- NULL for Mastodon-only login
    email VARCHAR(255) UNIQUE,
    
    -- Mastodon primary account
    primary_mastodon_instance VARCHAR(255),
    primary_mastodon_id VARCHAR(100),
    primary_mastodon_acct VARCHAR(200),
    
    -- ActivityPub fields
    private_key TEXT,
    public_key TEXT,
    actor_url VARCHAR(500) UNIQUE,
    inbox_url VARCHAR(500),
    outbox_url VARCHAR(500),
    followers_url VARCHAR(500),
    following_url VARCHAR(500),
    
    created_at TIMESTAMP DEFAULT NOW(),
    bio TEXT,
    avatar_url TEXT
);
```

#### posts
```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    content_html TEXT,
    is_anonymous BOOLEAN DEFAULT FALSE,
    
    -- ActivityPub
    ap_id VARCHAR(500) UNIQUE,
    ap_url VARCHAR(500),
    in_reply_to_id INTEGER REFERENCES posts(id),
    in_reply_to_ap_id VARCHAR(500),
    visibility VARCHAR(20) DEFAULT 'public',
    is_local BOOLEAN DEFAULT TRUE,
    remote_actor_id INTEGER REFERENCES ap_actors(id),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    published_at TIMESTAMP
);
```

#### ap_actors (remote ActivityPub users)
```sql
CREATE TABLE ap_actors (
    id SERIAL PRIMARY KEY,
    actor_url VARCHAR(500) UNIQUE NOT NULL,
    preferred_username VARCHAR(100),
    display_name VARCHAR(255),
    inbox_url VARCHAR(500) NOT NULL,
    outbox_url VARCHAR(500),
    public_key TEXT,
    avatar_url TEXT,
    summary TEXT,
    instance_url VARCHAR(255),
    last_fetched_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### follows
```sql
CREATE TABLE follows (
    id SERIAL PRIMARY KEY,
    follower_user_id INTEGER REFERENCES users(id),
    following_user_id INTEGER REFERENCES users(id),
    following_actor_id INTEGER REFERENCES ap_actors(id),
    ap_id VARCHAR(500) UNIQUE,
    state VARCHAR(20) DEFAULT 'accepted',
    created_at TIMESTAMP DEFAULT NOW(),
    
    CHECK (
        (following_user_id IS NOT NULL AND following_actor_id IS NULL) OR
        (following_user_id IS NULL AND following_actor_id IS NOT NULL)
    )
);
```

#### oauth_device_sessions (Device Flow)
```sql
CREATE TABLE oauth_device_sessions (
    id SERIAL PRIMARY KEY,
    device_code VARCHAR(100) UNIQUE NOT NULL,
    user_code VARCHAR(20) UNIQUE NOT NULL,
    verification_uri VARCHAR(500) NOT NULL,
    instance_url VARCHAR(255) NOT NULL,
    client_id VARCHAR(255),
    client_secret VARCHAR(255),
    expires_at TIMESTAMP NOT NULL,
    interval INTEGER DEFAULT 5,
    
    -- Post-authorization
    access_token TEXT,
    refresh_token TEXT,
    authorized_at TIMESTAMP,
    user_id INTEGER REFERENCES users(id),
    
    -- Mastodon account info
    mastodon_id VARCHAR(100),
    mastodon_username VARCHAR(100),
    mastodon_acct VARCHAR(200),
    
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### ap_activities (ActivityPub inbox/outbox)
```sql
CREATE TABLE ap_activities (
    id SERIAL PRIMARY KEY,
    activity_id VARCHAR(500) UNIQUE NOT NULL,
    activity_type VARCHAR(50) NOT NULL,  -- Create, Follow, Like, etc.
    actor_url VARCHAR(500) NOT NULL,
    object_id VARCHAR(500),
    object_type VARCHAR(50),
    target_id VARCHAR(500),
    activity_json JSONB NOT NULL,
    direction VARCHAR(10) NOT NULL,  -- inbox, outbox
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error TEXT
);
```

#### hashtags
```sql
CREATE TABLE hashtags (
    id SERIAL PRIMARY KEY,
    tag VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE post_hashtags (
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    hashtag_id INTEGER REFERENCES hashtags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, hashtag_id)
);
```

### Redis Schema

```
# Sessions
session:{session_id} -> {user_id, username, expires}

# Chatroulette queue
chatroulette:waiting -> SET of user_id/session_id

# Active chat sessions
chatroulette:session:{id} -> {user1, user2, started_at}

# Feed cache
feed:{user_id} -> LIST of post_id (last 100)

# Post cache
post:{post_id} -> JSON {content, user_id, votes, created_at}

# Trending hashtags
hashtag:trending -> ZSET (score=count, member=tag)

# Rate limiting
ratelimit:{ip}:post -> counter with TTL
ratelimit:{ip}:anon_post -> counter with TTL
```

## Authentication Flow (OAuth Device Flow)

### Step-by-Step Process

1. **User connects via SSH**
   ```
   $ ssh terminalpub.com
   ```

2. **Prompt for login method**
   ```
   [L] Login with Mastodon
   [A] Anonymous
   ```

3. **User chooses Login, enters instance**
   ```
   Mastodon instance: mastodon.social
   ```

4. **Server initiates Device Flow**
   - Register app on Mastodon instance (if not cached)
   - Request device code from Mastodon OAuth
   - Save session in `oauth_device_sessions`

5. **Display instructions to user**
   ```
   Visit: https://terminalpub.com/device
   Enter code: WXYZ-1234
   Waiting...
   ```

6. **User opens browser**
   - Goes to terminalpub.com/device
   - Enters code WXYZ-1234
   - Redirects to Mastodon OAuth
   - User authorizes on Mastodon
   - Callback to terminalpub.com/oauth/callback

7. **Server polls for authorization**
   - Every 5 seconds, check if token received
   - Once authorized, fetch Mastodon account info
   - Create or link local user account
   - Update session

8. **Login complete**
   ```
   ✓ Logged in as @alice@mastodon.social
   ```

### OAuth Endpoints

#### GET /.well-known/webfinger
- Discovery endpoint for ActivityPub
- Returns actor URL

#### GET /device
- Web page to enter device code
- Form submission redirects to Mastodon OAuth

#### GET /oauth/callback
- Receives OAuth callback from Mastodon
- Exchanges code for token
- Updates device session
- Shows success page

#### POST /api/v1/apps (on Mastodon instance)
- Registers terminalpub as OAuth app
- Returns client_id and client_secret
- Cached in database per instance

## ActivityPub Implementation

### Required Endpoints

#### GET /.well-known/webfinger
```json
{
  "subject": "acct:alice@terminalpub.com",
  "links": [{
    "rel": "self",
    "type": "application/activity+json",
    "href": "https://terminalpub.com/users/alice"
  }]
}
```

#### GET /users/{username}
```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "id": "https://terminalpub.com/users/alice",
  "type": "Person",
  "preferredUsername": "alice",
  "inbox": "https://terminalpub.com/users/alice/inbox",
  "outbox": "https://terminalpub.com/users/alice/outbox",
  "publicKey": {
    "id": "https://terminalpub.com/users/alice#main-key",
    "owner": "https://terminalpub.com/users/alice",
    "publicKeyPem": "-----BEGIN PUBLIC KEY-----\n..."
  }
}
```

#### POST /users/{username}/inbox
- Receives ActivityPub activities from other instances
- Validates HTTP signatures
- Queues for processing by inbox worker

#### GET /users/{username}/outbox
- Returns user's public posts as ActivityPub collection

#### GET /posts/{id}
- Returns single post as ActivityPub Note object

### Activity Types to Support

#### Create (Post)
```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Create",
  "actor": "https://terminalpub.com/users/alice",
  "object": {
    "type": "Note",
    "content": "Hello from terminalpub! #fediverse",
    "tag": [{
      "type": "Hashtag",
      "name": "#fediverse"
    }]
  }
}
```

#### Follow
```json
{
  "type": "Follow",
  "actor": "https://terminalpub.com/users/alice",
  "object": "https://mastodon.social/@bob"
}
```

#### Like (Upvote)
```json
{
  "type": "Like",
  "actor": "https://terminalpub.com/users/alice",
  "object": "https://mastodon.social/@bob/12345"
}
```

#### Announce (Boost/Repost)
```json
{
  "type": "Announce",
  "actor": "https://terminalpub.com/users/alice",
  "object": "https://mastodon.social/@bob/12345"
}
```

### HTTP Signatures

All ActivityPub requests must be signed:
```
Signature: keyId="https://terminalpub.com/users/alice#main-key",
           algorithm="rsa-sha256",
           headers="(request-target) host date digest",
           signature="..."
```

## User Interface (TUI)

### Main Menu (Authenticated)
```
╔════════════════════════════════╗
║     terminalpub                ║
║     @alice@mastodon.social     ║
╠════════════════════════════════╣
║  [F] Feed (23 new)             ║
║  [P] Post                      ║
║  [C] Chat Roulette             ║
║  [N] Notifications (3)         ║
║  [S] Search                    ║
║  [#] Trending Hashtags         ║
║  [M] Messages                  ║
║  [U] Profile                   ║
║  [Q] Quit                      ║
╚════════════════════════════════╝
```

### Feed View
```
╔══════════════════════════════════════════════╗
║  Feed                            [↑↓ scroll] ║
╠══════════════════════════════════════════════╣
║                                              ║
║  @bob@mastodon.social · 2h ago              ║
║  Just deployed my new app! 🚀 #golang       ║
║  ↑ 42  💬 5  🔁 12                           ║
║  ──────────────────────────────────────────  ║
║                                              ║
║  @carol@pixelfed.social · 4h ago            ║
║  Beautiful sunset today! #photography        ║
║  [image]                                     ║
║  ↑ 89  💬 12  🔁 23                          ║
║  ──────────────────────────────────────────  ║
║                                              ║
║  [Load more]                                 ║
╚══════════════════════════════════════════════╝

[r] reply  [u] upvote  [b] boost  [f] follow author
```

### Post Composer
```
╔══════════════════════════════════════════════╗
║  New Post                                    ║
╠══════════════════════════════════════════════╣
║                                              ║
║  ┌─────────────────────────────────────────┐║
║  │ What's on your mind?                    │║
║  │                                         │║
║  │ #hashtag support!                       │║
║  │ @mention@instance.com                   │║
║  │                                         │║
║  │ [280/500]                               │║
║  └─────────────────────────────────────────┘║
║                                              ║
║  Visibility: [Public ▼]                     ║
║  Post as: [@alice@mastodon.social ▼]        ║
║           [ ] Post anonymously              ║
║                                              ║
║  [Ctrl+Enter] Post  [Esc] Cancel            ║
╚══════════════════════════════════════════════╝
```

### Chat Roulette
```
╔══════════════════════════════════════════════╗
║  Chat Roulette                               ║
╠══════════════════════════════════════════════╣
║                                              ║
║  Stranger: Hey! Where are you from?         ║
║                                              ║
║  You: Hi! Italy. You?                       ║
║                                              ║
║  Stranger: Cool! I'm from Germany           ║
║                                              ║
║  > _                                         ║
║                                              ║
╠══════════════════════════════════════════════╣
║  [Esc] Next person  [Ctrl+C] Exit           ║
╚══════════════════════════════════════════════╝
```

## Features Specification

### 1. Anonymous Mode
- No login required
- Access to:
  - Public feed (global timeline)
  - Chat roulette
  - Anonymous posting (rate-limited by IP)
- No access to:
  - Personal feed
  - Following/followers
  - Notifications
  - Profile

### 2. Authenticated Mode
- Full access to all features
- Personal feed from followed accounts
- Post with attribution
- Follow/unfollow
- Upvote/comment
- Notifications
- Import Mastodon following/followers

### 3. Chat Roulette
- Random pairing of online users
- Anonymous conversations
- Either party can skip to next person
- Optional: Save conversation history (both must consent)
- Rate limiting to prevent abuse

### 4. Hashtag Support
- Parse hashtags from post content
- Store in separate table with relationship
- Clickable in TUI (mouse support via OSC 8)
- Trending hashtags page
- Search by hashtag

### 5. Federation
- Bidirectional sync with fediverse
- Follow Mastodon users from terminalpub
- Mastodon users can follow terminalpub users
- Posts visible across instances
- Likes/boosts federated
- Comments as ActivityPub replies

### 6. Import from Mastodon
- One-time import of following list
- One-time import of followers
- Optional: Import recent posts
- Sync running in background worker

## Configuration

### config.yaml Structure

```yaml
server:
  domain: terminalpub.com
  base_url: https://terminalpub.com
  ssh_port: 2222
  http_port: 8080
  https_port: 443
  tls:
    cert_file: /etc/terminalpub/cert.pem
    key_file: /etc/terminalpub/key.pem
    auto_cert: false

database:
  postgres:
    host: localhost
    port: 5432
    user: terminalpub
    password: ${POSTGRES_PASSWORD}
    database: terminalpub
    sslmode: disable
    max_connections: 25
  redis:
    host: localhost
    port: 6379
    password: ""
    db: 0

oauth:
  device_code_expiry: 600
  poll_interval: 5
  callback_url: https://terminalpub.com/oauth/callback

activitypub:
  enabled: true
  user_agent: "terminalpub/0.1.0"
  max_inbox_size: 1000
  delivery_workers: 10
  inbox_workers: 5
  retry_max_attempts: 5
  retry_base_delay: 30

features:
  chatroulette:
    enabled: true
    queue_timeout: 300
  anonymous_posting:
    enabled: true
    rate_limit: 10
  registration:
    enabled: true
    require_invite: false

security:
  rate_limiting:
    enabled: true
    requests_per_minute: 60
  blocked_instances:
    - spam.example.com

logging:
  level: info
  format: json
  output: stdout
```

## Development Phases

### Phase 1: Foundation (2 weeks)
- Project structure
- Go module setup
- Database schema
- Migrations
- Basic SSH server (echo server)
- Basic HTTP server

### Phase 2: Authentication (2 weeks)
- OAuth Device Flow implementation
- Mastodon app registration
- Token management
- Session handling
- Login/logout flows

### Phase 3: ActivityPub Core (2 weeks)
- WebFinger
- Actor endpoints
- HTTP signatures
- Inbox handler skeleton
- Outbox handler skeleton

### Phase 4: Core Features (2 weeks)
- Feed implementation
- Post creation
- Post display
- Follow/unfollow
- Upvote (Like)
- Comment (Reply)

### Phase 5: TUI Development (2 weeks)
- Main menu
- Feed view with scrolling
- Post composer
- Navigation
- Mouse support
- Hashtag clicking

### Phase 6: Federation (2 weeks)
- Inbox processor worker
- Delivery worker with retry
- Activity handlers (Create, Follow, Like, etc.)
- Remote actor fetching
- Federation debugging

### Phase 7: Social Features (2 weeks)
- Chat roulette
- Anonymous posting
- Hashtag parsing and storage
- Trending hashtags
- Search functionality
- Notifications

### Phase 8: Import & Sync (1 week)
- Mastodon following import
- Mastodon followers import
- Background sync worker

### Phase 9: Polish (2 weeks)
- Error handling
- Edge cases
- Rate limiting
- Moderation tools
- Performance optimization
- Security hardening

### Phase 10: Deployment (1 week)
- Production configuration
- Systemd services
- Monitoring
- Logging
- Backup strategy
- Documentation

**Total estimated time: ~16 weeks part-time**

## Security Considerations

### Authentication
- OAuth Device Flow (no password handling)
- Secure session tokens (random, long, expire)
- HTTPS required for OAuth callbacks
- Token refresh mechanism

### ActivityPub
- HTTP signature verification on all incoming activities
- Validate actor ownership
- Rate limiting per instance
- Instance blocklist support

### Input Validation
- Sanitize all user input
- Prevent XSS in HTML content
- SQL injection prevention (prepared statements)
- Validate ActivityPub JSON

### Rate Limiting
- Per-IP for anonymous actions
- Per-user for authenticated actions
- Per-instance for federation
- Exponential backoff for delivery retries

### Moderation
- Report system (future)
- Block users (future)
- Block instances
- Content warnings (future)

## Performance Targets

### Capacity
- 1000+ concurrent SSH connections
- 10,000+ registered users
- 100+ ActivityPub activities/second
- <100ms average response time
- <1s p99 response time

### Database
- Connection pooling (25 connections)
- Indexes on all foreign keys
- Indexes on common queries (user_id, created_at, ap_id)
- Redis caching for hot data

### Caching Strategy
- Feed: Last 100 posts per user (Redis)
- Posts: Individual post cache (Redis, 1 hour TTL)
- Actors: Remote actor cache (PostgreSQL, refresh daily)
- Sessions: Redis with TTL

### Monitoring
- Prometheus metrics
- Grafana dashboards
- Error tracking
- Performance profiling
- Activity queue depth

## Testing Strategy

### Unit Tests
- Models
- Services
- Utilities
- ActivityPub helpers

### Integration Tests
- Database operations
- Redis operations
- ActivityPub protocol
- OAuth flow

### E2E Tests
- SSH connection
- Login flow
- Post creation
- Federation scenarios

### Load Tests
- Concurrent SSH connections
- Feed rendering performance
- ActivityPub delivery throughput

## Deployment

### VPS Requirements
- **CPU**: 2 vCPUs
- **RAM**: 2GB minimum, 4GB recommended
- **Disk**: 20GB SSD minimum
- **OS**: Ubuntu 22.04 or Debian 12
- **Network**: Public IP, ports 22/2222, 443 open

### Services
- **PostgreSQL**: Docker container or native
- **Redis**: Docker container or native
- **terminalpub**: Systemd service
- **terminalpub-worker**: Systemd service
- **Nginx**: Reverse proxy for HTTPS (optional)

### Estimated Costs
- VPS (Hetzner CPX21): €5-7/month
- Domain: €10-15/year
- SSL (Let's Encrypt): Free
- **Total: ~€7/month**

## Repository Structure

```
terminalpub/
├── cmd/
│   ├── server/main.go
│   ├── worker/main.go
│   └── migrate/main.go
├── internal/
│   ├── activitypub/
│   ├── auth/
│   ├── config/
│   ├── db/
│   ├── handlers/
│   ├── models/
│   ├── services/
│   ├── ui/
│   └── workers/
├── migrations/
├── config/
├── web/
├── docs/
├── scripts/
├── tests/
├── .github/
├── README.md
├── LICENSE
├── Makefile
├── go.mod
└── docker-compose.yml
```

## Future Enhancements

### v1.1
- Notifications system
- Direct messages
- Profile customization
- Avatar support (ASCII art)

### v1.2
- Media attachments (images via URLs)
- Polls
- Content warnings
- Bookmarks

### v1.3
- Lists (custom feeds)
- Filters
- Mute/block
- Report system

### v2.0
- Multiple protocols (Nostr, AT Protocol?)
- Mobile app (separate project)
- Web interface (separate project)
- Federation statistics dashboard

## Contributing Guidelines

### Code Style
- `gofmt` and `goimports` required
- Follow Go standard conventions
- Comment exported functions
- Write tests for new features

### Commit Messages
- Use conventional commits
- Examples:
  - `feat: add chat roulette matching algorithm`
  - `fix: correct OAuth token refresh logic`
  - `docs: update ActivityPub implementation guide`

### Pull Requests
- One feature/fix per PR
- Include tests
- Update documentation
- Pass CI checks

## License

AGPLv3 - All network services using this code must release their source code.

---

**Document Version**: 1.0  
**Last Updated**: 2025-11-22  
**Maintained by**: @fulgidus
