#!/bin/bash

# ================================================================
# Redis Security Setup Script
# ================================================================
# Purpose: Generate strong password and configure Redis securely
# Date: 2026-03-22
# ================================================================

set -e  # Exit on error

echo "========================================="
echo "Redis Security Setup"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ================================================================
# 1. Generate Strong Password
# ================================================================
echo -e "${YELLOW}[1/5] Generating strong password...${NC}"

if command -v openssl &> /dev/null; then
    REDIS_PASSWORD=$(openssl rand -base64 32)
    echo -e "${GREEN}✓ Password generated successfully${NC}"
else
    echo -e "${RED}✗ openssl not found. Please install openssl first.${NC}"
    exit 1
fi

echo ""
echo "Generated Password:"
echo "==================="
echo "$REDIS_PASSWORD"
echo "==================="
echo ""
echo -e "${YELLOW}⚠️  IMPORTANT: Save this password securely!${NC}"
echo ""

# ================================================================
# 2. Update redis.conf
# ================================================================
echo -e "${YELLOW}[2/5] Updating redis.conf...${NC}"

REDIS_CONF="../redis.conf"

if [ -f "$REDIS_CONF" ]; then
    # Backup original file
    cp "$REDIS_CONF" "${REDIS_CONF}.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${GREEN}✓ Backup created${NC}"
    
    # Replace password in redis.conf
    sed -i "s/requirepass CHANGE_THIS_TO_STRONG_PASSWORD/requirepass $REDIS_PASSWORD/" "$REDIS_CONF"
    echo -e "${GREEN}✓ redis.conf updated${NC}"
else
    echo -e "${RED}✗ redis.conf not found at $REDIS_CONF${NC}"
    exit 1
fi

# ================================================================
# 3. Update .env file
# ================================================================
echo -e "${YELLOW}[3/5] Updating .env file...${NC}"

ENV_FILE="../.env"

if [ -f "$ENV_FILE" ]; then
    # Backup original file
    cp "$ENV_FILE" "${ENV_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${GREEN}✓ Backup created${NC}"
    
    # Update REDIS_PASSWORD in .env
    if grep -q "^REDIS_PASSWORD=" "$ENV_FILE"; then
        sed -i "s/^REDIS_PASSWORD=.*/REDIS_PASSWORD=$REDIS_PASSWORD/" "$ENV_FILE"
        echo -e "${GREEN}✓ .env updated${NC}"
    else
        echo "REDIS_PASSWORD=$REDIS_PASSWORD" >> "$ENV_FILE"
        echo -e "${GREEN}✓ REDIS_PASSWORD added to .env${NC}"
    fi
else
    echo -e "${RED}✗ .env file not found at $ENV_FILE${NC}"
    exit 1
fi

# ================================================================
# 4. Create Redis directories
# ================================================================
echo -e "${YELLOW}[4/5] Creating Redis directories...${NC}"

sudo mkdir -p /var/lib/redis
sudo mkdir -p /var/log/redis
sudo chown -R redis:redis /var/lib/redis 2>/dev/null || echo "Note: Redis user not found, skipping chown"
sudo chown -R redis:redis /var/log/redis 2>/dev/null || echo "Note: Redis user not found, skipping chown"

echo -e "${GREEN}✓ Directories created${NC}"

# ================================================================
# 5. Instructions
# ================================================================
echo ""
echo -e "${YELLOW}[5/5] Next Steps:${NC}"
echo "==========================================
echo ""
echo "1. Restart Redis with the new configuration:"
echo "   sudo systemctl restart redis"
echo "   OR"
echo "   redis-server /path/to/redis.conf"
echo ""
echo "2. Test the connection:"
echo "   redis-cli -a $REDIS_PASSWORD ping"
echo "   Expected output: PONG"
echo ""
echo "3. Verify security settings:"
echo "   redis-cli -a $REDIS_PASSWORD CONFIG GET bind"
echo "   Expected: 127.0.0.1"
echo ""
echo "4. Update your application code if needed"
echo ""
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Redis Security Setup Complete!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "${YELLOW}⚠️  Remember to:${NC}"
echo "   - Keep the password secure"
echo "   - Update any scripts or tools that connect to Redis"
echo "   - Test your application thoroughly"
echo ""
