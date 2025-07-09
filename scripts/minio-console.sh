#!/bin/bash

# MinIO Console Access Script
# This script helps access the MinIO console securely via SSH tunnel

set -e

# Load environment variables
if [ -f .env ]; then
    source .env
fi

echo "🔐 MinIO Console Access Helper"
echo "================================"
echo ""
echo "The MinIO console is not exposed publicly for security."
echo "Choose your access method:"
echo ""
echo "1. SSH Tunnel (recommended for remote access)"
echo "2. Direct container access (local server only)"
echo "3. MinIO Client (mc) commands"
echo ""
read -p "Select option (1-3): " choice

case $choice in
    1)
        echo ""
        echo "🔒 Setting up SSH tunnel..."
        echo "Run this command on your LOCAL machine:"
        echo ""
        echo "ssh -L 9001:localhost:9001 user@your-server.com"
        echo ""
        echo "Then access MinIO console at: http://localhost:9001"
        echo "Username: ${MINIO_ROOT_USER:-admin}"
        echo "Password: ${MINIO_ROOT_PASSWORD:-[check your .env file]}"
        ;;
    2)
        echo ""
        echo "🖥️  Direct container access..."
        echo "MinIO console is available at container port 9001"
        
        # Get container IP
        CONTAINER_IP=$(docker inspect butler-minio | grep -E '"IPAddress".*[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -1 | cut -d'"' -f4)
        
        if [ -n "$CONTAINER_IP" ]; then
            echo "Container IP: $CONTAINER_IP"
            echo "Access at: http://$CONTAINER_IP:9001"
        else
            echo "❌ Could not determine container IP"
        fi
        
        echo ""
        echo "Or use port forwarding:"
        echo "docker-compose exec minio sh -c 'echo \"Access console at http://localhost:9001\"'"
        ;;
    3)
        echo ""
        echo "📱 MinIO Client (mc) commands..."
        echo "Use these commands for admin tasks:"
        echo ""
        echo "# List buckets"
        echo "docker-compose exec minio mc ls"
        echo ""
        echo "# Bucket info"
        echo "docker-compose exec minio mc stat butler-storage"
        echo ""
        echo "# User management"
        echo "docker-compose exec minio mc admin user list"
        echo ""
        echo "# Server info"
        echo "docker-compose exec minio mc admin info"
        echo ""
        echo "# Policy management"
        echo "docker-compose exec minio mc admin policy list"
        ;;
    *)
        echo "❌ Invalid option"
        exit 1
        ;;
esac

echo ""
echo "🔐 Security Note:"
echo "The MinIO console provides full admin access to your storage."
echo "Never expose it publicly - always use secure access methods."