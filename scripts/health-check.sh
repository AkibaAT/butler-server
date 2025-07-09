#!/bin/bash

# Butler Server Health Check Script
set -e

# Load environment variables
if [ -f .env ]; then
    source .env
fi

echo "🏥 Butler Server Health Check"
echo "================================"

# Check if services are running
echo "📊 Checking service status..."
if ! docker-compose ps | grep -q "Up"; then
    echo "❌ Some services are not running"
    docker-compose ps
    exit 1
fi

# Check Butler Server health
echo "🔍 Checking Butler Server..."
if curl -f -s "https://${BUTLER_SUBDOMAIN}.${DOMAIN}/" > /dev/null; then
    echo "✅ Butler Server is healthy"
else
    echo "❌ Butler Server is not responding"
    exit 1
fi

# Check Butler API health
echo "🔍 Checking Butler API..."
if curl -f -s "https://${BUTLER_API_SUBDOMAIN}.${DOMAIN}/" > /dev/null; then
    echo "✅ Butler API is healthy"
else
    echo "❌ Butler API is not responding"
    exit 1
fi

# Check MinIO health
echo "🔍 Checking MinIO..."
if curl -f -s "https://${BUTLER_STORAGE_SUBDOMAIN}.${DOMAIN}/minio/health/live" > /dev/null; then
    echo "✅ MinIO is healthy"
else
    echo "❌ MinIO is not responding"
    exit 1
fi

# Check Database health
echo "🔍 Checking Database..."
if docker-compose exec -T db pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB} > /dev/null; then
    echo "✅ Database is healthy"
else
    echo "❌ Database is not responding"
    exit 1
fi

echo ""
echo "🎉 All services are healthy!"
echo "📊 Service URLs:"
echo "  - Butler Server: https://${BUTLER_SUBDOMAIN}.${DOMAIN}"
echo "  - Butler API: https://${BUTLER_API_SUBDOMAIN}.${DOMAIN}"
echo "  - MinIO Storage: https://${BUTLER_STORAGE_SUBDOMAIN}.${DOMAIN}"
echo "  - MinIO Console: SSH tunnel required (port 9001)"