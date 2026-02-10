#!/bin/bash
set -e

echo "🚀 Starting KrillinAI Manual Deployment..."

# 0. Define Variables
NETWORK_NAME="krillin_net"
REDIS_NAME="krillin_redis"
APP_NAME="krillin"
IMAGE_NAME="asteria798/krillinai"
DOCKER_CMD="/Applications/Docker.app/Contents/Resources/bin/docker"

# 1. Create Network (if not exists)
echo "🌐 Setting up network..."
if ! $DOCKER_CMD network ls | grep -q "$NETWORK_NAME"; then
    $DOCKER_CMD network create "$NETWORK_NAME"
    echo "   Network created."
else
    echo "   Network exists."
fi

# 2. Setup Redis
echo "📦 Setting up Redis..."
if $DOCKER_CMD ps -a | grep -q "$REDIS_NAME"; then
    echo "   Removing old Redis container..."
    $DOCKER_CMD rm -f "$REDIS_NAME"
fi
$DOCKER_CMD run -d \
    --name "$REDIS_NAME" \
    --network "$NETWORK_NAME" \
    --restart always \
    redis:alpine
echo "   Redis started."

# 3. Build Application
echo "🏗️  Building KrillinAI image (this may take a while)..."
# Use optimized Dockerfile.local
$DOCKER_CMD build -f Dockerfile.local -t "$IMAGE_NAME" .

# 4. Stop Old Application
echo "🛑 Stopping old application..."
if $DOCKER_CMD ps -a | grep -q "$APP_NAME"; then
    $DOCKER_CMD rm -f "$APP_NAME"
fi

# 5. Start Application
echo "🚀 Starting KrillinAI..."
$DOCKER_CMD run -d \
    --name "$APP_NAME" \
    --network "$NETWORK_NAME" \
    -p 8888:8888 \
    -e REDIS_ADDR="$REDIS_NAME:6379" \
    -v "$(pwd)/config/config.toml:/app/config/config.toml" \
    -v "$(pwd)/tasks:/app/tasks" \
    -v "$(pwd)/models:/app/models" \
    -v "$(pwd)/bin:/app/bin" \
    -v "$(pwd)/cookies.txt:/app/cookies.txt" \
    -v "$(pwd)/static:/app/static" \
    -v "$(pwd)/summary.txt:/app/summary.txt" \
    --restart always \
    "$IMAGE_NAME"

echo "✅ Deployment Complete!"
echo "📜 Tailing logs (Ctrl+C to exit)..."
sleep 2
$DOCKER_CMD logs -f "$APP_NAME"
