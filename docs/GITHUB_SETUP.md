# GitHub Actions Setup Instructions

## Configure VPS_SSH_KEY Secret

To enable automatic deployment from GitHub Actions to your VPS, you need to add the SSH private key as a GitHub Secret.

### Step 1: Copy the Private Key

The private key has been generated at: `~/.ssh/terminalpub_deploy`

Copy the entire content including the BEGIN and END lines:

```bash
cat ~/.ssh/terminalpub_deploy
```

### Step 2: Add Secret to GitHub

1. Go to your repository on GitHub
2. Navigate to: **Settings** → **Secrets and variables** → **Actions**
3. Click **"New repository secret"**
4. Set the following:
   - **Name**: `VPS_SSH_KEY`
   - **Value**: Paste the entire private key content
5. Click **"Add secret"**

### Step 3: Verify Setup

After adding the secret, you can verify the setup by:

1. Push a commit to the main branch:
   ```bash
   git push origin main
   ```

2. Go to the **Actions** tab in your GitHub repository

3. Watch the deployment workflow run

4. If successful, the server will be deployed to: `51.91.97.241:2222`

### What Happens During Deployment

The GitHub Action will:
1. ✅ Run all tests
2. ✅ Build the Go binary for Linux AMD64
3. ✅ Connect to VPS using SSH
4. ✅ Stop any old SSH experiment server
5. ✅ Install terminalpub as a systemd service
6. ✅ Start the service
7. ✅ Run health check

### Testing the Deployed Server

Once deployed, you can connect to the server:

```bash
ssh 51.91.97.241 -p 2222
```

### Troubleshooting

**If deployment fails:**

1. Check GitHub Actions logs:
   - Go to repository → Actions tab
   - Click on the failed workflow
   - Expand the failed step to see error details

2. Verify the secret is set:
   - Settings → Secrets and variables → Actions
   - You should see `VPS_SSH_KEY` in the list

3. Test SSH connection manually:
   ```bash
   ssh -i ~/.ssh/terminalpub_deploy -p 2222 ubuntu@51.91.97.241
   ```

4. Check VPS logs:
   ```bash
   ssh -p 2222 ubuntu@51.91.97.241 'sudo journalctl -u terminalpub -n 50'
   ```

### Manual Deployment

If you need to deploy manually without GitHub Actions:

```bash
# Build the binary
make build

# Deploy using the script
scp -P 2222 -i ~/.ssh/terminalpub_deploy bin/terminalpub ubuntu@51.91.97.241:/tmp/
scp -P 2222 -i ~/.ssh/terminalpub_deploy scripts/deploy.sh ubuntu@51.91.97.241:/tmp/
scp -P 2222 -i ~/.ssh/terminalpub_deploy scripts/terminalpub.service ubuntu@51.91.97.241:/tmp/
ssh -p 2222 -i ~/.ssh/terminalpub_deploy ubuntu@51.91.97.241 'bash /tmp/deploy.sh'
```

## Security Notes

- ✅ The public key has already been added to the VPS
- ✅ The private key should ONLY be added to GitHub Secrets (never commit it!)
- ✅ The deployment key has no passphrase for automated deployment
- ✅ The key is dedicated only for deployment purposes

## Next Steps

After successful deployment:

1. Connect to your server: `ssh 51.91.97.241 -p 2222`
2. Start implementing Phase 2 features (Authentication)
3. Monitor server logs: `ssh -p 2222 ubuntu@51.91.97.241 'sudo journalctl -u terminalpub -f'`
