# Butler Server Production Deployment

This guide covers deploying Butler Server in a production environment with Docker, PostgreSQL, MinIO, and Traefik.

## Prerequisites

- Docker and Docker Compose installed
- Traefik reverse proxy running with Docker provider enabled
- External network `web` created for Traefik
- Domain `fvn.li` configured with DNS pointing to your server
- Cloudflare SSL certificates configured in Traefik

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd butler-server
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Deploy**
   ```bash
   ./deploy.sh
   ```

## Environment Configuration

### Required Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Domain Configuration
DOMAIN=fvn.li
BUTLER_SUBDOMAIN=butler
BUTLER_API_SUBDOMAIN=api.butler
BUTLER_STORAGE_SUBDOMAIN=storage.butler

# Database Configuration
POSTGRES_DB=butler
POSTGRES_USER=butler
POSTGRES_PASSWORD=your_secure_password_here

# MinIO Configuration
MINIO_ACCESS_KEY=your_minio_access_key
MINIO_SECRET_KEY=your_minio_secret_key
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=your_minio_root_password
```

### Service URLs

After deployment, services will be available at:

- **Butler Server**: `https://butler.fvn.li`
- **Butler API**: `https://api.butler.fvn.li`
- **MinIO Storage**: `https://storage.butler.fvn.li`
- **MinIO Console**: Available via SSH tunnel only (see Security section)

## Traefik Configuration

The docker-compose.yml includes Traefik labels for:

- **Docker provider**: Uses labels for service discovery
- **External network**: Connects to existing `web` network
- **SSL termination**: Handled by Cloudflare → Traefik
- **Subdomain routing**: Each service gets its own subdomain

## Security Considerations

- **Non-root containers**: All services run as non-root users
- **Health checks**: Configured for all services
- **Secrets management**: Use Docker secrets in production
- **Network isolation**: Services communicate on internal network
- **SSL/TLS**: Terminated at Cloudflare with source certificates
- **MinIO Console**: Not exposed publicly - use SSH tunnel for admin access

### MinIO Console Access (SSH Tunnel)

The MinIO console is not exposed publicly for security. To access it:

1. **Create SSH tunnel**:
   ```bash
   ssh -L 9001:localhost:9001 user@your-server.com
   ```

2. **Access console locally**:
   ```bash
   # Find the MinIO container IP
   docker inspect butler-minio | grep IPAddress
   
   # Or use docker exec to access
   docker-compose exec minio curl http://localhost:9001
   ```

3. **Alternative - Direct container access**:
   ```bash
   # Access MinIO console through container
   docker-compose exec minio mc admin info local
   ```

This approach keeps the admin interface secure while still allowing necessary administrative access.

## Volumes and Data Persistence

- **PostgreSQL**: `butler_postgres_data`
- **MinIO**: `butler_minio_data`

Volumes are automatically created and persist between deployments.

## Monitoring and Maintenance

### Check service status
```bash
docker-compose ps
```

### View logs
```bash
docker-compose logs -f [service-name]
```

### Update deployment
```bash
git pull
./deploy.sh
```

### Backup database
```bash
docker-compose exec db pg_dump -U butler butler > backup.sql
```

### Backup MinIO data
```bash
docker-compose exec minio mc mirror /data /backup
```

## Troubleshooting

### Service not accessible
1. Check Traefik logs: `docker logs traefik`
2. Verify DNS resolution
3. Check service health: `docker-compose ps`

### Database connection issues
1. Check PostgreSQL logs: `docker-compose logs db`
2. Verify credentials in `.env`
3. Check network connectivity

### MinIO issues
1. Check MinIO logs: `docker-compose logs minio`
2. Verify bucket creation
3. Test API endpoints

## Production Optimizations

### Resource Limits
Add to docker-compose.yml:
```yaml
deploy:
  resources:
    limits:
      cpus: '1.0'
      memory: 1G
    reservations:
      cpus: '0.5'
      memory: 512M
```

### Log Rotation
Configure in `/etc/docker/daemon.json`:
```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

### Monitoring
Consider adding:
- Prometheus metrics
- Grafana dashboards
- Alertmanager for notifications
- Health check endpoints

## Support

For issues and questions:
- Check service logs
- Review Traefik configuration
- Verify network connectivity
- Check DNS resolution