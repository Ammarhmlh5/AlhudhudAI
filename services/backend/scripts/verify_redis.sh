#!/bin/bash

# ================================================================
# Redis Security Verification Script
# ================================================================
# Purpose: Verify that Redis is properly secured
# Date: 2026-03-22
# ================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Redis Security Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# ================================================================
# Get Redis password from .env
# ================================================================
if [ -f "../.env" ]; then
    REDIS_PASSWORD=$(grep "^REDIS_PASSWORD=" ../.env | cut -d '=' -f2)
    if [ -z "$REDIS_PASSWORD" ]; then
        echo -e "${RED}✗ REDIS_PASSWORD is empty in .env${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ .env file not found${NC}"
    exit 1
fi

# ================================================================
# Test 1: Connection without password (should fail)
# ================================================================
echo -e "${YELLOW}[1/8] Testing connection without password...${NC}"
if redis-cli ping 2>&1 | grep -q "NOAUTH"; then
    echo -e "${GREEN}✓ Password protection is working${NC}"
else
    echo -e "${RED}✗ Redis is NOT password protected!${NC}"
    exit 1
fi
echo ""

# ================================================================
# Test 2: Connection with password (should succeed)
# ================================================================
echo -e "${YELLOW}[2/8] Testing connection with password...${NC}"
if redis-cli -a "$REDIS_PASSWORD" ping 2>&1 | grep -q "PONG"; then
    echo -e "${GREEN}✓ Authentication successful${NC}"
else
    echo -e "${RED}✗ Authentication failed${NC}"
    exit 1
fi
echo ""

# ================================================================
# Test 3: Verify bind address
# ================================================================
echo -e "${YELLOW}[3/8] Checking bind address...${NC}"
BIND_ADDRESS=$(redis-cli -a "$REDIS_PASSWORD" CONFIG GET bind 2>/dev/null | tail -n 1)
if [ "$BIND_ADDRESS" = "127.0.0.1" ]; then
    echo -e "${GREEN}✓ Bind address is 127.0.0.1 (secure)${NC}"
else
    echo -e "${RED}✗ Bind address is: $BIND_ADDRESS (insecure!)${NC}"
fi
echo ""

# ================================================================
# Test 4: Verify dangerous commands are disabled
# ================================================================
echo -e "${YELLOW}[4/8] Checking dangerous commands...${NC}"

DANGEROUS_COMMANDS=("FLUSHALL" "FLUSHDB" "CONFIG" "SHUTDOWN" "DEBUG" "KEYS")
ALL_DISABLED=true

for cmd in "${DANGEROUS_COMMANDS[@]}"; do
    if redis-cli -a "$REDIS_PASSWORD" $cmd 2>&1 | grep -q "unknown command"; then
        echo -e "${GREEN}✓ $cmd is disabled${NC}"
    else
        echo -e "${RED}✗ $cmd is NOT disabled!${NC}"
        ALL_DISABLED=false
    fi
done
echo ""

# ================================================================
# Test 5: Check maxmemory setting
# ================================================================
echo -e "${YELLOW}[5/8] Checking maxmemory setting...${NC}"
MAXMEMORY=$(redis-cli -a "$REDIS_PASSWORD" CONFIG GET maxmemory 2>/dev/null | tail -n 1)
if [ "$MAXMEMORY" != "0" ]; then
    MAXMEMORY_MB=$((MAXMEMORY / 1024 / 1024))
    echo -e "${GREEN}✓ maxmemory is set to ${MAXMEMORY_MB}MB${NC}"
else
    echo -e "${YELLOW}⚠ maxmemory is not set (unlimited)${NC}"
fi
echo ""

# ================================================================
# Test 6: Check maxmemory-policy
# ================================================================
echo -e "${YELLOW}[6/8] Checking maxmemory-policy...${NC}"
POLICY=$(redis-cli -a "$REDIS_PASSWORD" CONFIG GET maxmemory-policy 2>/dev/null | tail -n 1)
if [ "$POLICY" = "allkeys-lru" ]; then
    echo -e "${GREEN}✓ maxmemory-policy is allkeys-lru${NC}"
else
    echo -e "${YELLOW}⚠ maxmemory-policy is: $POLICY${NC}"
fi
echo ""

# ================================================================
# Test 7: Check AOF persistence
# ================================================================
echo -e "${YELLOW}[7/8] Checking AOF persistence...${NC}"
AOF=$(redis-cli -a "$REDIS_PASSWORD" CONFIG GET appendonly 2>/dev/null | tail -n 1)
if [ "$AOF" = "yes" ]; then
    echo -e "${GREEN}✓ AOF persistence is enabled${NC}"
else
    echo -e "${YELLOW}⚠ AOF persistence is disabled${NC}"
fi
echo ""

# ================================================================
# Test 8: Get Redis info
# ================================================================
echo -e "${YELLOW}[8/8] Getting Redis info...${NC}"
echo "----------------------------"
redis-cli -a "$REDIS_PASSWORD" INFO server 2>/dev/null | grep -E "redis_version|os|tcp_port|uptime_in_days"
echo ""
redis-cli -a "$REDIS_PASSWORD" INFO memory 2>/dev/null | grep -E "used_memory_human|maxmemory_human"
echo ""
redis-cli -a "$REDIS_PASSWORD" INFO persistence 2>/dev/null | grep -E "aof_enabled|rdb_last_save_time"
echo ""

# ================================================================
# Summary
# ================================================================
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Verification Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

SCORE=0
MAX_SCORE=8

# Check 1: Password protection
if redis-cli ping 2>&1 | grep -q "NOAUTH"; then
    ((SCORE++))
fi

# Check 2: Authentication works
if redis-cli -a "$REDIS_PASSWORD" ping 2>&1 | grep -q "PONG"; then
    ((SCORE++))
fi

# Check 3: Bind address
if [ "$BIND_ADDRESS" = "127.0.0.1" ]; then
    ((SCORE++))
fi

# Check 4: Dangerous commands
if [ "$ALL_DISABLED" = true ]; then
    ((SCORE++))
fi

# Check 5: Maxmemory
if [ "$MAXMEMORY" != "0" ]; then
    ((SCORE++))
fi

# Check 6: Maxmemory policy
if [ "$POLICY" = "allkeys-lru" ]; then
    ((SCORE++))
fi

# Check 7: AOF
if [ "$AOF" = "yes" ]; then
    ((SCORE++))
fi

# Check 8: Redis is running
if redis-cli -a "$REDIS_PASSWORD" ping 2>&1 | grep -q "PONG"; then
    ((SCORE++))
fi

PERCENTAGE=$((SCORE * 100 / MAX_SCORE))

echo "Score: $SCORE/$MAX_SCORE ($PERCENTAGE%)"
echo ""

if [ $SCORE -eq $MAX_SCORE ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✓ Redis Security: EXCELLENT${NC}"
    echo -e "${GREEN}========================================${NC}"
elif [ $SCORE -ge 6 ]; then
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}⚠ Redis Security: GOOD (minor issues)${NC}"
    echo -e "${YELLOW}========================================${NC}"
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}✗ Redis Security: NEEDS IMPROVEMENT${NC}"
    echo -e "${RED}========================================${NC}"
fi

echo ""
echo "Next Steps:"
echo "1. Review any warnings above"
echo "2. Update redis.conf if needed"
echo "3. Restart Redis: sudo systemctl restart redis"
echo "4. Run this script again to verify"
echo ""
