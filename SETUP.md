# Setup Instructions for terminalpub

## Local Development Setup

### Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose
- Git

### Initial Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/fulgidus/terminalpub
   cd terminalpub
   ```

2. **Install dependencies**
   ```bash
   make install-deps
   ```

3. **Start local databases**
   ```bash
   make docker-up
   ```

4. **Run the server**
   ```bash
   make dev
   ```

5. **Connect via SSH**
   ```bash
   ssh localhost -p 2222
   ```

## GitHub Secrets Configuration

For automatic deployment to work, you need to configure the following secrets in your GitHub repository:

### Required Secret: VPS_SSH_KEY

1. **Generate SSH key pair** (if you don't have one):
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/terminalpub_deploy -C "github-deploy"
   ```

2. **Add public key to VPS**:
   ```bash
   # Copy the public key
   cat ~/.ssh/terminalpub_deploy.pub
   
   # On the VPS, add it to authorized_keys
   ssh ubuntu@51.91.97.241 -p 2222
   echo "YOUR_PUBLIC_KEY" >> ~/.ssh/authorized_keys
   ```

3. **Add private key to GitHub Secrets**:
   - Go to: https://github.com/fulgidus/terminalpub/settings/secrets/actions
   - Click "New repository secret"
   - Name: `VPS_SSH_KEY`
   - Value: Paste the content of `~/.ssh/terminalpub_deploy` (the PRIVATE key)
   - Click "Add secret"

### Testing the Deployment

After setting up the secret, push to main branch:

```bash
git add .
git commit -m "feat: initial project setup"
git push origin main
```

The GitHub Action will:
1. Run tests
2. Build the binary
3. Deploy to VPS at 51.91.97.241
4. Stop any old SSH server
5. Install and start terminalpub as a systemd service

## VPS Manual Setup (if needed)

If you need to manually deploy or troubleshoot:

```bash
# Connect to VPS
ssh ubuntu@51.91.97.241 -p 2222

# Check service status
sudo systemctl status terminalpub

# View logs
sudo journalctl -u terminalpub -f

# Restart service
sudo systemctl restart terminalpub

# Stop service
sudo systemctl stop terminalpub
```

## Development Commands

- `make help` - Show all available commands
- `make build` - Build the binary
- `make run` - Build and run
- `make dev` - Run with hot reload
- `make test` - Run tests
- `make lint` - Run linter
- `make format` - Format code
- `make clean` - Clean build artifacts

## Project Structure

```
terminalpub/
├── cmd/
│   ├── server/          # Main SSH+HTTP server
│   ├── worker/          # Background federation worker
│   └── migrate/         # Database migration tool
├── internal/
│   ├── activitypub/     # ActivityPub protocol
│   ├── auth/            # Authentication & OAuth
│   ├── db/              # Database layer
│   ├── handlers/        # Request handlers
│   ├── models/          # Data models
│   ├── services/        # Business logic
│   ├── ui/              # TUI components
│   └── workers/         # Background jobs
├── migrations/          # SQL migrations
├── config/              # Configuration files
├── scripts/             # Deployment scripts
└── .github/workflows/   # CI/CD configuration
```

## Next Steps

1. ✅ Project structure initialized
2. ✅ Basic SSH server created
3. ✅ CI/CD pipeline configured
4. ✅ Deployment scripts ready
5. 🔄 Configure GitHub secrets (see above)
6. 🔄 Push to main to trigger first deployment
7. 🔜 Implement authentication (Phase 2)
8. 🔜 Implement ActivityPub (Phase 3)
9. 🔜 Build core features (Phase 4+)

## Troubleshooting

### Build Issues
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

### Connection Issues
```bash
# Check if server is running
ps aux | grep terminalpub

# Check SSH port
netstat -tlnp | grep 2222
```

### VPS Deployment Issues
```bash
# Check GitHub Actions logs
# Go to: https://github.com/fulgidus/terminalpub/actions

# Manual deployment
scp -P 2222 bin/terminalpub ubuntu@51.91.97.241:/tmp/
ssh -p 2222 ubuntu@51.91.97.241 'bash /tmp/deploy.sh'
```
