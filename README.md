# Simple Butler Server

A minimal butler-compatible server implementation for hosting your own game distribution platform. Compatible with the itch.io butler command-line tool.

## Features

- **Butler CLI compatibility**: Works with the existing butler push/fetch commands
- **Wharf patch system**: Supports incremental updates and patch creation
- **Local file storage**: Simple local filesystem storage for game files
- **SQLite database**: Lightweight database for metadata storage
- **Resumable uploads**: Supports chunked uploads for large files
- **User authentication**: API key-based authentication

## Quick Start

### 1. Build and Run

```bash
# Clone/copy the server code
cd butler-server

# Build
go build -o butler-server

# Create a test user
./butler-server -create-user=myusername

# Start the server
./butler-server
```

### 2. Configure Butler

```bash
# Point butler to your server
butler --address=http://localhost:8080 login

# When prompted for OAuth, it will automatically create a test user
```

### 3. Push Your First Game

```bash
# Create a test game directory
mkdir my-game
echo "Hello World" > my-game/game.txt

# Push to your server
butler --address=http://localhost:8080 push my-game myusername/my-game:windows
```

## Usage

### Server Commands

```bash
# Start server (default port 8080)
./butler-server

# Custom port and paths
./butler-server -port=9090 -db=./my-games.db -storage=./my-files

# Create a user
./butler-server -create-user=username
```

### Butler Commands

```bash
# Set server address for all commands
export BUTLER_API_SERVER=http://localhost:8080

# Or use the --address flag
butler --address=http://localhost:8080 <command>

# Login (creates test user automatically)
butler login

# Push a game
butler push /path/to/game username/gamename:channel

# Check status
butler status username/gamename:channel

# List your games (via API)
curl "http://localhost:8080/profile/games?api_key=YOUR_API_KEY"
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
GET  /uploads/{id}/download     # Get download URL
GET  /builds/{id}               # Get build info
```

### Wharf API (Butler Push)

```
GET  /wharf/status                           # Check status
GET  /wharf/channels/{channel}               # Get channel info
POST /wharf/builds                           # Create build
GET  /wharf/builds/{id}/files               # List build files
POST /wharf/builds/{id}/files               # Create build file
POST /wharf/builds/{buildId}/files/{fileId} # Finalize file
GET  /wharf/builds/{buildId}/files/{fileId}/download # Download file
```

### Upload/Download

```
POST /upload/{sessionId}                    # Upload file chunks
GET  /downloads/builds/{buildId}/files/{fileId}     # Download build file
GET  /downloads/uploads/{uploadId}/{filename}       # Download upload
```

## Configuration

### Environment Variables

- `BUTLER_API_SERVER`: Server URL for butler commands
- `BUTLER_API_KEY`: API key for authentication

### Server Flags

- `-port`: Server port (default: 8080)
- `-db`: SQLite database path (default: ./butler-server.db)
- `-storage`: File storage directory (default: ./storage)
- `-create-user`: Create a test user and exit

## Database Schema

The server uses SQLite with these main tables:

- **users**: User accounts and API keys
- **games**: Game metadata
- **uploads**: File uploads for games
- **builds**: Wharf builds (versions)
- **build_files**: Individual files within builds
- **channels**: Distribution channels (e.g., "windows", "linux")
- **upload_sessions**: Resumable upload tracking

## File Storage

Files are stored locally in the storage directory:

```
storage/
├── builds/
│   ├── 1/          # Build ID 1
│   │   ├── patch_default_uuid1
│   │   └── signature_default_uuid2
│   └── 2/          # Build ID 2
└── uploads/
    └── 1/          # Upload ID 1
        └── game.zip
```

## Development

### Testing with curl

```bash
# Get server info
curl http://localhost:8080/

# Create user (server must be running)
./butler-server -create-user=testuser

# Test API (replace with your API key)
API_KEY="your-api-key-here"
curl "http://localhost:8080/profile?api_key=$API_KEY"

# Test game creation via butler
butler --address=http://localhost:8080 push ./test-game testuser/test-game:windows
```

### Adding Features

The server is designed to be extensible:

1. **models/**: Add new database models
2. **handlers/**: Add new API endpoints
3. **storage/**: Extend storage backends (S3, GCS, etc.)
4. **auth/**: Add new authentication methods

## Limitations

This is a simple implementation for development/testing:

- **No user registration UI**: Users are created via command line
- **Basic authentication**: Only API key auth, no OAuth flow
- **Local storage only**: No cloud storage integration
- **No admin interface**: Database management via SQL only
- **Minimal error handling**: Basic error responses
- **No rate limiting**: No protection against abuse
- **No HTTPS**: Development only, add reverse proxy for production

## Production Considerations

For production use, consider:

1. **HTTPS**: Use nginx/caddy as reverse proxy
2. **Database**: PostgreSQL for better concurrency
3. **Storage**: S3/GCS for scalability
4. **Authentication**: Proper OAuth implementation
5. **Monitoring**: Add logging and metrics
6. **Security**: Input validation, rate limiting
7. **Backup**: Database and file backup strategy

## License

MIT License - see LICENSE file for details.