#!/bin/bash

# Butler Server Production Deployment Script
set -e

echo "🚀 Starting Butler Server deployment..."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ .env file not found. Please copy .env.example to .env and configure it."
    exit 1
fi

# Load environment variables
source .env

echo "📦 Building Butler Server image..."
docker-compose build --no-cache butler-server

echo "🗄️ Creating volumes if they don't exist..."
docker volume create ${DB_VOLUME_NAME} 2>/dev/null || true
docker volume create ${MINIO_VOLUME_NAME} 2>/dev/null || true

echo "🔧 Stopping existing containers..."
docker-compose down

echo "🚀 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if services are healthy
echo "🏥 Checking service health..."
if docker-compose ps | grep -q "unhealthy\|exited"; then
    echo "❌ Some services are not healthy. Check logs:"
    docker-compose logs --tail=50
    exit 1
fi

echo "✅ Deployment completed successfully!"
echo ""
echo "🌐 Services available at:"
echo "  - Butler Server: https://${BUTLER_SUBDOMAIN}.${DOMAIN}"
echo "  - Butler API: https://${BUTLER_API_SUBDOMAIN}.${DOMAIN}"
echo "  - MinIO Storage: https://${BUTLER_STORAGE_SUBDOMAIN}.${DOMAIN}"
echo "  - MinIO Console: SSH tunnel to port 9001 (secure admin access)"
echo ""
echo "📋 To view logs: docker-compose logs -f"
echo "📋 To stop services: docker-compose down"