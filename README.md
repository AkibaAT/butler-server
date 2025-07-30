# Butler Server

## What is this?

Butler Server is a self-hosted game distribution server that works with the itch.io butler CLI tool. It allows developers to run their own game distribution infrastructure while maintaining full compatibility with existing butler workflows.

**⚠️ Early Development Warning**

This project is in **very early proof-of-concept** stage. The basic functionality works, but consider this experimental code. It's not ready for production use yet, and things might break or change as development continues. Great for experimenting and testing though!

## What it does

- **Butler CLI support**: `butler push/status/channels` commands work, `fetch` has known issues
- **Build versioning**: Parent build tracking for future patch support
- **MinIO storage**: S3-compatible storage with secure downloads
- **PostgreSQL database**: Reliable database backend
- **Direct uploads**: Files go straight to storage via presigned URLs
- **User isolation**: Users can only access their own games
- **Admin access**: Admins can manage any namespace
- **DDEV setup**: Streamlined development environment (requires `ddev start` + build step)

## Quick Start

### Get it running

```bash
# Clone and start
git clone <repository-url>
cd butler-server
ddev start

# Build and create a user
ddev exec "go build -o butler-server ."
ddev exec "./butler-server --create-user=myusername"

# Start the server
ddev exec "./butler-server"
```

### Use butler with it

```bash
# Point butler at your server
butler --address=https://butler-server.ddev.site login
# (paste the API key from when you created the user)

# Push a game
mkdir my-game
echo "Hello World v1.0" > my-game/game.txt
butler --address=https://butler-server.ddev.site push my-game myusername/my-game:main

# Check it worked
butler --address=https://butler-server.ddev.site status myusername/my-game:main
```

## Usage

### Managing users

```bash
# Create users
ddev exec "./butler-server --create-user=alice"      # Regular user
ddev exec "./butler-server --create-admin=admin"     # Admin user

# See who exists
ddev exec "./butler-server --list-users"

# Enable/disable users
ddev exec "./butler-server --activate-user=alice"
ddev exec "./butler-server --deactivate-user=alice"
```

### Butler commands

```bash
# Set this once to avoid typing --address every time
export BUTLER_API_SERVER=https://butler-server.ddev.site

# Login with your API key
butler login

# Push games (format: username/gamename:channel)
butler push /path/to/game username/gamename:main
butler push /path/to/game username/gamename:beta

# Check what's up
butler status username/gamename:main
butler channels username/gamename

# Note: butler fetch has known issues in current version
```

### Direct API calls

```bash
# Grab your API key from user creation, then:
API_KEY="your-api-key-here"

curl -H "Authorization: $API_KEY" https://butler-server.ddev.site/wharf/status
curl -H "Authorization: $API_KEY" https://butler-server.ddev.site/profile/games
```

## API Endpoints

### Core API

```
GET  /profile                    # Get user profile
GET  /profile/games             # List user's games
GET  /games/{id}                # Get game info
GET  /games/{id}/uploads        # List game uploads
GET  /uploads/{id}              # Get upload info
GET  /uploads/{id}/builds       # List upload builds
GET  /builds/{id}               # Get build info
```

### Wharf API (Butler Compatible)

```
GET  /wharf/status                                    # Check server status
GET  /wharf/channels                                  # List all channels
GET  /wharf/channels/{channel}                        # Get channel info
POST /wharf/builds                                    # Create new build
GET  /wharf/builds/{id}/files                        # List build files
POST /wharf/builds/{id}/files                        # Create build file (get upload URL)
POST /wharf/builds/{buildId}/files/{fileId}          # Finalize uploaded file
GET  /wharf/builds/{buildId}/files/{fileId}/download  # Get download redirect
```

### File Upload/Download Flow

1. **Upload**: Client calls `POST /wharf/builds/{id}/files` → Gets presigned MinIO upload URL → Uploads directly to MinIO → Calls finalize endpoint
2. **Download**: Client calls download endpoint → Server returns redirect to signed MinIO URL → Client downloads directly from MinIO

**Direct storage**: Files go straight to/from MinIO using signed URLs - no server bottlenecks.

## Configuration

### Environment Variables

**Database (PostgreSQL):**
- `POSTGRES_HOST`: Database host (default: `db` in DDEV)
- `POSTGRES_PORT`: Database port (default: `5432`)
- `POSTGRES_DB`: Database name (default: `db`)
- `POSTGRES_USER`: Database user (default: `db`)
- `POSTGRES_PASSWORD`: Database password (default: `db`)

**Storage (MinIO):**
- `MINIO_ENDPOINT`: MinIO endpoint (default: `minio:9000` in DDEV)
- `MINIO_ACCESS_KEY`: MinIO access key (default: `ddevminio`)
- `MINIO_SECRET_KEY`: MinIO secret key (default: `ddevminio`)
- `MINIO_BUCKET`: Storage bucket name (default: `butler-storage`)
- `MINIO_USE_SSL`: Use SSL for MinIO (default: `false`)

**Butler Client:**
- `BUTLER_API_SERVER`: Server URL for butler commands
- `BUTLER_API_KEY`: API key for authentication

### Server Flags

- `--port`: Server port (default: `8080`)
- `--create-user=username`: Create a regular user and exit
- `--create-admin=username`: Create an admin user and exit
- `--list-users`: List all users and exit
- `--activate-user=username`: Activate user account
- `--deactivate-user=username`: Deactivate user account

## Architecture

### Database Schema (PostgreSQL)

The server uses PostgreSQL with these main tables:

- **users**: User accounts, API keys, and roles (`user`/`admin`)
- **games**: Game metadata with namespace ownership
- **uploads**: Upload metadata (files stored in MinIO)
- **builds**: Wharf builds (versions) with state tracking
- **build_files**: Individual files within builds (stored in MinIO)
- **channels**: Distribution channels (`main`, `beta`, etc.)

### File Storage (MinIO S3)

Files are stored in MinIO object storage with authenticated access:

```
butler-storage/           # MinIO bucket
├── builds/
│   ├── 1/               # Build ID 1
│   │   └── archive_default_uuid1.zip
│   └── 2/               # Build ID 2
│       └── archive_default_uuid2.zip
└── test/                # Test files
    └── hello.txt
```

**How files work:**
- Private bucket (no public access)
- Upload URLs expire in 1 hour
- Download URLs expire in 1 hour
- Files go directly to/from MinIO (server doesn't proxy)

### How security works

- **Regular users**: Can only access `username/*` games
- **Admin users**: Can access any namespace, but games stay owned by the original user
- **User isolation**: Users can't see each other's stuff
- **Ownership**: Games belong to the namespace owner, not whoever created them

## Development

### DDEV Environment

The project includes a complete DDEV configuration:

```yaml
# .ddev/config.yaml
name: butler-server
type: generic
docroot: .
php_version: "8.3"
nodejs_version: "22"
database:
  type: postgres
  version: "17"
```

**Services:**
- **Web**: Generic container with Go 1.24+ and butler CLI
- **Database**: PostgreSQL 17 with automatic migrations
- **MinIO**: S3-compatible storage with web UI at https://butler-server.ddev.site:9090

### Testing

```bash
# Test server status
curl -s https://butler-server.ddev.site/test/minio | jq .

# Test API with authentication
API_KEY="your-api-key-here"
curl -H "Authorization: $API_KEY" https://butler-server.ddev.site/wharf/status

# Test butler workflow (push and status work, fetch has issues)
mkdir test-game && echo "Hello World" > test-game/game.txt
butler --address=https://butler-server.ddev.site push test-game testuser/test-game:main
butler --address=https://butler-server.ddev.site status testuser/test-game:main
```

### Database Access

```bash
# Connect to PostgreSQL
ddev exec "psql -h db -U db -d db"

# View current data
ddev exec "psql -h db -U db -d db -c 'SELECT u.username, g.title FROM games g JOIN users u ON g.user_id = u.id;'"
```

### Adding Features

The server is designed to be extensible:

1. **models/**: Database models and interfaces
2. **handlers/**: HTTP handlers for API endpoints
3. **auth/**: Authentication and authorization
4. **migrations/**: Database schema changes

## Why it's built this way

- **PostgreSQL database**: Reliable multi-user database backend
- **MinIO/S3 storage**: Better than local files, works with CDNs and scales nicely
- **User isolation**: People can't mess with each other's games
- **Direct file transfers**: Fast uploads/downloads, server doesn't get in the way
- **Good error handling**: Won't crash on weird requests
- **Request logging**: You can see what's happening
- **HTTPS ready**: Works behind nginx, cloudflare, whatever you use

## Running it for real

### Docker

```bash
# Build it
docker build -t butler-server .

# Run it (you'll need postgres and minio somewhere)
docker run -d \
  -e POSTGRES_HOST=your-postgres-host \
  -e POSTGRES_USER=butler \
  -e POSTGRES_PASSWORD=secure-password \
  -e MINIO_ENDPOINT=your-minio-endpoint \
  -e MINIO_ACCESS_KEY=your-access-key \
  -e MINIO_SECRET_KEY=your-secret-key \
  -p 8080:8080 \
  butler-server
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: butler-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: butler-server
  template:
    metadata:
      labels:
        app: butler-server
    spec:
      containers:
      - name: butler-server
        image: butler-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: POSTGRES_HOST
          value: "postgres-service"
        - name: MINIO_ENDPOINT
          value: "minio-service:9000"
        # Add other environment variables
```

### Scaling it

- **Database**: PostgreSQL handles lots of connections
- **Storage**: MinIO clusters or just use AWS S3
- **Stateless**: Servers don't store anything, spin up as many as you want
- **CDN**: Stick CloudFront in front of MinIO for global downloads

## Security stuff

### Users and permissions

- **API keys**: Each user gets a token
- **Roles**: Regular users vs admins
- **Namespaces**: Users can only touch their own games
- **Ownership**: Games belong to the namespace, not who created them

### File security

- **Private bucket**: No public access to MinIO
- **Timed URLs**: Upload/download URLs expire in 1 hour
- **Signed URLs**: Can't be faked or modified
- **No direct access**: Everything goes through authentication

### Other security

- **PostgreSQL**: Proper database with foreign keys
- **Environment config**: No passwords in code
- **Request logging**: See who's doing what
- **Input validation**: Won't crash on garbage input

## Checking if it's working

### Health checks

```bash
# Is the server up?
curl https://butler-server.ddev.site/wharf/status

# Can it talk to the database?
curl -H "Authorization: $API_KEY" https://butler-server.ddev.site/profile

# Can it talk to MinIO?
curl https://butler-server.ddev.site/test/minio
```

### What gets logged

The server logs:
- All HTTP requests (method, path, response code)
- Authentication attempts
- Errors with stack traces
- Database queries (if you want)

### Database queries

```sql
-- How many connections?
SELECT count(*) FROM pg_stat_activity;

-- How big are the tables?
SELECT schemaname,tablename,pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables WHERE schemaname='public';

-- What's been happening lately?
SELECT g.title, u.username, b.created_at
FROM builds b
JOIN uploads up ON b.upload_id = up.id
JOIN games g ON up.game_id = g.id
JOIN users u ON g.user_id = u.id
ORDER BY b.created_at DESC LIMIT 10;
```

## License

MIT License - see LICENSE file for details.