#!/bin/bash
# Fix PostgreSQL permissions for terminalpub user
# Run this on the VPS as root or sudo user

set -e

echo "======================================"
echo "Fixing PostgreSQL Permissions"
echo "======================================"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Applying permission fixes...${NC}"

# Run the SQL script as postgres user
sudo -u postgres psql -d terminalpub << 'EOF'
-- Grant usage on schema
GRANT USAGE ON SCHEMA public TO terminalpub;

-- Grant permissions on all existing tables
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO terminalpub;

-- Grant permissions on all sequences (for auto-increment IDs)
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO terminalpub;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO terminalpub;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO terminalpub;

-- Verify permissions
SELECT grantee, privilege_type, table_name 
FROM information_schema.role_table_grants 
WHERE grantee = 'terminalpub'
ORDER BY table_name, privilege_type;
EOF

echo -e "${GREEN}Permissions fixed successfully!${NC}"
echo ""
echo -e "${YELLOW}Restarting terminalpub service...${NC}"
sudo systemctl restart terminalpub

sleep 2

if systemctl is-active --quiet terminalpub; then
    echo -e "${GREEN}✓ terminalpub service is running${NC}"
else
    echo -e "${RED}✗ terminalpub service failed to start${NC}"
    echo "Check logs with: sudo journalctl -u terminalpub -n 50"
    exit 1
fi

echo ""
echo -e "${GREEN}======================================"
echo "Permission fix completed!"
echo "======================================${NC}"
