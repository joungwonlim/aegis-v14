#!/bin/bash

# Aegis v14 API Server Restart Script
# Usage: ./restart-api.sh

echo "🔄 Restarting Aegis v14 API Server..."

# Kill existing processes on port 8099
echo "🔪 Killing processes on port 8099..."
lsof -ti:8099 | xargs kill -9 2>/dev/null

if [ $? -eq 0 ]; then
    echo "✅ Processes killed"
else
    echo "ℹ️  No processes found on port 8099"
fi

# Wait a moment for port to be released
sleep 1

# Start API server
echo "🚀 Starting API server..."
cd "$(dirname "$0")"
go run ./cmd/api/main.go
