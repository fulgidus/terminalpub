#!/bin/bash
set -e

echo "================================"
echo "Deploying terminalpub to VPS"
echo "================================"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="terminalpub"
APP_USER="terminalpub"
APP_DIR="/opt/terminalpub"
BINARY_PATH="/tmp/terminalpub"
SERVICE_FILE="/tmp/terminalpub.service"

echo -e "${YELLOW}Step 1: Stopping old SSH experiment...${NC}"
# Check if there's an old SSH server running
if systemctl is-active --quiet old-ssh-server 2>/dev/null; then
    sudo systemctl stop old-ssh-server
    sudo systemctl disable old-ssh-server
    echo -e "${GREEN}Old SSH server stopped${NC}"
fi

# Check for any custom SSH services on port 22 (besides system sshd)
if [ -f /etc/systemd/system/ssh-experiment.service ]; then
    sudo systemctl stop ssh-experiment || true
    sudo systemctl disable ssh-experiment || true
    sudo rm -f /etc/systemd/system/ssh-experiment.service
    echo -e "${GREEN}Removed SSH experiment service${NC}"
fi

echo -e "${YELLOW}Step 2: Creating application user and directories...${NC}"
# Create application user if it doesn't exist
if ! id "$APP_USER" &>/dev/null; then
    sudo useradd -r -s /bin/false -d $APP_DIR $APP_USER
    echo -e "${GREEN}Created user: $APP_USER${NC}"
fi

# Create application directory
sudo mkdir -p $APP_DIR/{bin,config,logs,data,.ssh}
sudo chown -R $APP_USER:$APP_USER $APP_DIR

echo -e "${YELLOW}Step 3: Stopping existing terminalpub service...${NC}"
# Stop existing service if running
if systemctl is-active --quiet $APP_NAME 2>/dev/null; then
    sudo systemctl stop $APP_NAME
    echo -e "${GREEN}Service stopped${NC}"
fi

echo -e "${YELLOW}Step 4: Installing new binary...${NC}"
# Move binary to application directory
sudo mv $BINARY_PATH $APP_DIR/bin/$APP_NAME
sudo chmod +x $APP_DIR/bin/$APP_NAME
sudo chown $APP_USER:$APP_USER $APP_DIR/bin/$APP_NAME
echo -e "${GREEN}Binary installed${NC}"

echo -e "${YELLOW}Step 5: Generating SSH host key...${NC}"
# Generate SSH host key if it doesn't exist
if [ ! -f $APP_DIR/.ssh/term_ed25519 ]; then
    sudo -u $APP_USER ssh-keygen -t ed25519 -f $APP_DIR/.ssh/term_ed25519 -N "" -C "terminalpub-host-key"
    echo -e "${GREEN}SSH host key generated${NC}"
else
    echo -e "${GREEN}SSH host key already exists${NC}"
fi

echo -e "${YELLOW}Step 6: Installing systemd service...${NC}"
# Install systemd service
sudo mv $SERVICE_FILE /etc/systemd/system/$APP_NAME.service
sudo systemctl daemon-reload
sudo systemctl enable $APP_NAME
echo -e "${GREEN}Service installed${NC}"

echo -e "${YELLOW}Step 7: Starting terminalpub service...${NC}"
sudo systemctl start $APP_NAME

# Wait a moment for the service to start
sleep 2

echo -e "${YELLOW}Step 8: Checking service status...${NC}"
if systemctl is-active --quiet $APP_NAME; then
    echo -e "${GREEN}✓ terminalpub is running!${NC}"
    sudo systemctl status $APP_NAME --no-pager -l
else
    echo -e "${RED}✗ Failed to start terminalpub${NC}"
    sudo journalctl -u $APP_NAME -n 50 --no-pager
    exit 1
fi

echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}Deployment completed successfully!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo "Service: $APP_NAME"
echo "Status: sudo systemctl status $APP_NAME"
echo "Logs: sudo journalctl -u $APP_NAME -f"
echo "SSH: ssh localhost -p 2222"
echo ""
