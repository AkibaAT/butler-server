#!/bin/bash

# Butler Server Backup Script
set -e

# Load environment variables
if [ -f .env ]; then
    source .env
fi

BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "💾 Starting Butler Server backup..."
echo "📁 Backup directory: $BACKUP_DIR"

# Backup PostgreSQL database
echo "🗃️ Backing up PostgreSQL database..."
docker-compose exec -T db pg_dump -U ${POSTGRES_USER} ${POSTGRES_DB} > "$BACKUP_DIR/database.sql"
echo "✅ Database backup completed"

# Backup MinIO data
echo "📦 Backing up MinIO data..."
docker-compose exec -T minio mc mirror --overwrite /data "$BACKUP_DIR/minio/"
echo "✅ MinIO backup completed"

# Backup configuration files
echo "⚙️ Backing up configuration..."
cp .env "$BACKUP_DIR/env.backup"
cp docker-compose.yml "$BACKUP_DIR/"
echo "✅ Configuration backup completed"

# Create compressed archive
echo "🗜️ Creating compressed archive..."
tar -czf "$BACKUP_DIR.tar.gz" -C backups "$(basename $BACKUP_DIR)"
rm -rf "$BACKUP_DIR"
echo "✅ Backup archive created: $BACKUP_DIR.tar.gz"

# Cleanup old backups (keep last 7 days)
echo "🧹 Cleaning up old backups..."
find backups/ -name "*.tar.gz" -type f -mtime +7 -delete
echo "✅ Cleanup completed"

echo ""
echo "🎉 Backup completed successfully!"
echo "📁 Backup file: $BACKUP_DIR.tar.gz"
echo "💡 To restore, extract the archive and run restore.sh"