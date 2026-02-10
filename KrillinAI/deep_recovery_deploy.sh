#!/bin/bash
set -e

echo "============================================"
echo "🔧 DEEP DOCKER RECOVERY & DEPLOYMENT SCRIPT"
echo "============================================"
echo ""

DOCKER_CMD="/Applications/Docker.app/Contents/Resources/bin/docker"

# Step 1: Kill ALL stuck Docker CLI processes
echo "🛑 Step 1: Killing stuck Docker CLI processes..."
pkill -9 -f "docker build" || true
pkill -9 -f "docker ps" || true
pkill -9 -f "docker info" || true
pkill -9 -f "docker stats" || true
pkill -9 -f "docker network" || true
pkill -9 -f "docker-buildx" || true
sleep 2
echo "   ✅ Done."

# Step 2: Quit Docker Desktop gracefully
echo "🛑 Step 2: Quitting Docker Desktop..."
osascript -e 'quit app "Docker Desktop"' || true
osascript -e 'quit app "Docker"' || true
sleep 5

# Step 3: Force kill if still running
echo "🛑 Step 3: Force killing remaining Docker processes..."
pkill -9 -f "com.docker" || true
pkill -9 -f "Docker Desktop" || true
sleep 3
echo "   ✅ Done."

# Step 4: Clean Docker socket (sometimes stale)
echo "🧹 Step 4: Cleaning Docker socket files..."
rm -f ~/Library/Containers/com.docker.docker/Data/docker.sock 2>/dev/null || true
rm -f /var/run/docker.sock 2>/dev/null || true
echo "   ✅ Done."

# Step 5: Restart Docker Desktop
echo "🚀 Step 5: Starting Docker Desktop..."
open -a "Docker"
echo "   Waiting for Docker to become ready (this may take 30-60 seconds)..."

# Step 6: Wait for Docker daemon to be ready
MAX_WAIT=120
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    if $DOCKER_CMD info > /dev/null 2>&1; then
        echo "   ✅ Docker is ready!"
        break
    fi
    sleep 5
    WAITED=$((WAITED + 5))
    echo "   Still waiting... ($WAITED seconds)"
done

if [ $WAITED -ge $MAX_WAIT ]; then
    echo "❌ Docker failed to start after $MAX_WAIT seconds."
    echo "Please try rebooting your computer."
    exit 1
fi

# Step 7: Verify Docker is working
echo "🔍 Step 6: Verifying Docker..."
$DOCKER_CMD version
echo "   ✅ Docker verified."

# Step 8: Proceed with deployment
echo ""
echo "============================================"
echo "🚀 STARTING DEPLOYMENT"
echo "============================================"
echo ""

# Network setup
NETWORK_NAME="krillin_net"
echo "🌐 Setting up network..."
if ! $DOCKER_CMD network ls --format '{{.Name}}' | grep -q "^${NETWORK_NAME}$"; then
    $DOCKER_CMD network create "$NETWORK_NAME"
    echo "   Network created."
else
    echo "   Network exists."
fi

# Redis setup
REDIS_NAME="krillin_redis"
echo "📦 Setting up Redis..."
$DOCKER_CMD rm -f "$REDIS_NAME" 2>/dev/null || true
$DOCKER_CMD run -d \
    --name "$REDIS_NAME" \
    --network "$NETWORK_NAME" \
    --restart always \
    redis:alpine
echo "   ✅ Redis started."

# Build application
echo "🏗️ Building KrillinAI (this takes several minutes)..."
$DOCKER_CMD build -f Dockerfile.local -t asteria798/krillinai . --progress=plain

# Stop and remove old app container
APP_NAME="krillin"
echo "🛑 Stopping old application..."
$DOCKER_CMD rm -f "$APP_NAME" 2>/dev/null || true

# Start application
echo "🚀 Starting KrillinAI..."
$DOCKER_CMD run -d \
    --name "$APP_NAME" \
    --network "$NETWORK_NAME" \
    -p 8888:8888 \
    -e REDIS_ADDR="${REDIS_NAME}:6379" \
    -v "$(pwd)/config/config.toml:/app/config/config.toml" \
    -v "$(pwd)/tasks:/app/tasks" \
    -v "$(pwd)/models:/app/models" \
    -v "$(pwd)/bin:/app/bin" \
    -v "$(pwd)/cookies.txt:/app/cookies.txt" \
    -v "$(pwd)/static:/app/static" \
    --restart always \
    asteria798/krillinai

echo ""
echo "============================================"
echo "✅ DEPLOYMENT COMPLETE!"
echo "============================================"
echo ""
echo "Application is running at: http://localhost:8888"
echo ""
echo "📜 Tailing logs (Ctrl+C to exit)..."
sleep 3
$DOCKER_CMD logs -f "$APP_NAME"
