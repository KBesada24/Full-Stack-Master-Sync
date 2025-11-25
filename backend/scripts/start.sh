#!/bin/bash
# Full Stack Master Sync Backend - Startup Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Full Stack Master Sync Backend${NC}"

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  No .env file found. Creating from .env.example...${NC}"
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo -e "${GREEN}✅ Created .env file from .env.example${NC}"
        echo -e "${YELLOW}⚠️  Please update .env with your configuration${NC}"
    else
        echo -e "${RED}❌ No .env.example file found. Please create .env manually.${NC}"
        exit 1
    fi
fi

# Load environment variables
export $(grep -v '^#' .env | xargs)

# Set defaults if not provided
export PORT=${PORT:-8080}
export ENVIRONMENT=${ENVIRONMENT:-development}
export LOG_LEVEL=${LOG_LEVEL:-info}

echo -e "${GREEN}📋 Configuration:${NC}"
echo "   Port: $PORT"
echo "   Environment: $ENVIRONMENT"
echo "   Log Level: $LOG_LEVEL"

# Check if binary exists, if not build it
if [ ! -f "full-stack-sync-backend" ]; then
    echo -e "${YELLOW}📦 Building application...${NC}"
    go build -o full-stack-sync-backend .
    echo -e "${GREEN}✅ Build complete${NC}"
fi

# Start the server
echo -e "${GREEN}🌐 Starting server on port $PORT...${NC}"
./full-stack-sync-backend
