package models

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// User database methods
func (d *SQLiteDatabase) GetUserByAPIKey(apiKey string) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, username, display_name, api_key, role, is_active, created_at, updated_at
		FROM users WHERE api_key = ? AND is_active = 1`, apiKey).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *SQLiteDatabase) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, username, display_name, api_key, role, is_active, created_at, updated_at
		FROM users WHERE id = ?`, id).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *SQLiteDatabase) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, username, display_name, api_key, role, is_active, created_at, updated_at
		FROM users WHERE username = ?`, username).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *SQLiteDatabase) CreateUser(user *User) error {
	// Set default values if not provided
	if user.Role == "" {
		user.Role = "user"
	}
	if !user.IsActive {
		user.IsActive = true
	}

	result, err := d.db.Exec(`
		INSERT INTO users (username, display_name, api_key, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		user.Username, user.DisplayName, user.APIKey, user.Role, user.IsActive)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (d *SQLiteDatabase) UpdateUser(user *User) error {
	_, err := d.db.Exec(`
		UPDATE users SET username = ?, display_name = ?, api_key = ?, role = ?, is_active = ?, updated_at = datetime('now')
		WHERE id = ?`,
		user.Username, user.DisplayName, user.APIKey, user.Role, user.IsActive, user.ID)
	return err
}

func (d *SQLiteDatabase) ListUsers() ([]*User, error) {
	rows, err := d.db.Query(`
		SELECT id, username, display_name, api_key, role, is_active, created_at, updated_at
		FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.APIKey,
			&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// Game database methods
func (d *SQLiteDatabase) GetGameByID(id int64) (*User, *Game, error) {
	var user User
	var game Game

	err := d.db.QueryRow(`
		SELECT
			g.id, g.user_id, g.title, g.short_text, g.type, g.classification, g.url, g.created_at, g.updated_at,
			u.id, u.username, u.display_name, u.api_key, u.role, u.is_active, u.created_at, u.updated_at
		FROM games g
		JOIN users u ON g.user_id = u.id
		WHERE g.id = ?`, id).Scan(
		&game.ID, &game.UserID, &game.Title, &game.ShortText, &game.Type, &game.Classification, &game.URL, &game.CreatedAt, &game.UpdatedAt,
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, nil, err
	}

	return &user, &game, nil
}

func (d *SQLiteDatabase) GetGamesByUserID(userID int64) ([]*Game, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, title, short_text, type, classification, url, created_at, updated_at
		FROM games WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*Game
	for rows.Next() {
		game := &Game{}
		err := rows.Scan(&game.ID, &game.UserID, &game.Title, &game.ShortText,
			&game.Type, &game.Classification, &game.URL, &game.CreatedAt, &game.UpdatedAt)
		if err != nil {
			return nil, err
		}
		games = append(games, game)
	}
	return games, nil
}

func (d *SQLiteDatabase) GetGameByUserAndTitle(userID int64, title string) (*Game, error) {
	game := &Game{}
	err := d.db.QueryRow(`
		SELECT id, user_id, title, short_text, type, classification, url, created_at, updated_at
		FROM games WHERE user_id = ? AND title = ?`, userID, title).Scan(
		&game.ID, &game.UserID, &game.Title, &game.ShortText,
		&game.Type, &game.Classification, &game.URL, &game.CreatedAt, &game.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return game, nil
}

func (d *SQLiteDatabase) CreateGame(game *Game) error {
	result, err := d.db.Exec(`
		INSERT INTO games (user_id, title, short_text, type, classification, url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		game.UserID, game.Title, game.ShortText, game.Type, game.Classification, game.URL)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	game.ID = id
	return nil
}

// Upload database methods
func (d *SQLiteDatabase) GetUploadByID(id int64) (*Upload, error) {
	upload := &Upload{}
	err := d.db.QueryRow(`
		SELECT id, game_id, filename, display_name, size, storage, type, platforms, created_at, updated_at
		FROM uploads WHERE id = ?`, id).Scan(
		&upload.ID, &upload.GameID, &upload.Filename, &upload.DisplayName,
		&upload.Size, &upload.Storage, &upload.Type, &upload.Platforms,
		&upload.CreatedAt, &upload.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return upload, nil
}

func (d *SQLiteDatabase) GetUploadsByGameID(gameID int64) ([]*Upload, error) {
	rows, err := d.db.Query(`
		SELECT id, game_id, filename, display_name, size, storage, type, platforms, created_at, updated_at
		FROM uploads WHERE game_id = ?`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uploads []*Upload
	for rows.Next() {
		upload := &Upload{}
		err := rows.Scan(&upload.ID, &upload.GameID, &upload.Filename, &upload.DisplayName,
			&upload.Size, &upload.Storage, &upload.Type, &upload.Platforms,
			&upload.CreatedAt, &upload.UpdatedAt)
		if err != nil {
			return nil, err
		}
		uploads = append(uploads, upload)
	}
	return uploads, nil
}

func (d *SQLiteDatabase) CreateUpload(upload *Upload) error {
	result, err := d.db.Exec(`
		INSERT INTO uploads (game_id, filename, display_name, size, storage, type, platforms, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		upload.GameID, upload.Filename, upload.DisplayName, upload.Size,
		upload.Storage, upload.Type, upload.Platforms)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	upload.ID = id
	return nil
}

// Build database methods
func (d *SQLiteDatabase) GetBuildByID(id int64) (*Build, error) {
	build := &Build{}
	var parentBuildID sql.NullInt64

	err := d.db.QueryRow(`
		SELECT id, upload_id, user_version, parent_build_id, state, created_at, updated_at
		FROM builds WHERE id = ?`, id).Scan(
		&build.ID, &build.UploadID, &build.UserVersion, &parentBuildID,
		&build.State, &build.CreatedAt, &build.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if parentBuildID.Valid {
		build.ParentBuildID = &parentBuildID.Int64
	}

	return build, nil
}

func (d *SQLiteDatabase) GetBuildsByUploadID(uploadID int64) ([]*Build, error) {
	rows, err := d.db.Query(`
		SELECT id, upload_id, user_version, parent_build_id, state, created_at, updated_at
		FROM builds WHERE upload_id = ? ORDER BY id DESC`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var builds []*Build
	for rows.Next() {
		build := &Build{}
		var parentBuildID sql.NullInt64

		err := rows.Scan(&build.ID, &build.UploadID, &build.UserVersion, &parentBuildID,
			&build.State, &build.CreatedAt, &build.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if parentBuildID.Valid {
			build.ParentBuildID = &parentBuildID.Int64
		}

		builds = append(builds, build)
	}
	return builds, nil
}

func (d *SQLiteDatabase) CreateBuild(build *Build) error {
	var parentBuildID interface{}
	if build.ParentBuildID != nil {
		parentBuildID = *build.ParentBuildID
	}

	result, err := d.db.Exec(`
		INSERT INTO builds (upload_id, user_version, parent_build_id, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
		build.UploadID, build.UserVersion, parentBuildID, build.State)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	build.ID = id
	return nil
}

func (d *SQLiteDatabase) UpdateBuild(build *Build) error {
	var parentBuildID interface{}
	if build.ParentBuildID != nil {
		parentBuildID = *build.ParentBuildID
	}

	_, err := d.db.Exec(`
		UPDATE builds SET upload_id = ?, user_version = ?, parent_build_id = ?, state = ?, updated_at = datetime('now')
		WHERE id = ?`,
		build.UploadID, build.UserVersion, parentBuildID, build.State, build.ID)
	return err
}

// BuildFile database methods
func (d *SQLiteDatabase) GetBuildFileByID(id int64) (*BuildFile, error) {
	buildFile := &BuildFile{}
	err := d.db.QueryRow(`
		SELECT id, build_id, type, sub_type, size, state, storage_path, upload_url, created_at, updated_at
		FROM build_files WHERE id = ?`, id).Scan(
		&buildFile.ID, &buildFile.BuildID, &buildFile.Type, &buildFile.SubType,
		&buildFile.Size, &buildFile.State, &buildFile.StoragePath, &buildFile.UploadURL,
		&buildFile.CreatedAt, &buildFile.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return buildFile, nil
}

func (d *SQLiteDatabase) GetBuildFilesByBuildID(buildID int64) ([]*BuildFile, error) {
	rows, err := d.db.Query(`
		SELECT id, build_id, type, sub_type, size, state, storage_path, upload_url, created_at, updated_at
		FROM build_files WHERE build_id = ?`, buildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buildFiles []*BuildFile
	for rows.Next() {
		buildFile := &BuildFile{}
		err := rows.Scan(&buildFile.ID, &buildFile.BuildID, &buildFile.Type, &buildFile.SubType,
			&buildFile.Size, &buildFile.State, &buildFile.StoragePath, &buildFile.UploadURL,
			&buildFile.CreatedAt, &buildFile.UpdatedAt)
		if err != nil {
			return nil, err
		}
		buildFiles = append(buildFiles, buildFile)
	}
	return buildFiles, nil
}

func (d *SQLiteDatabase) CreateBuildFile(buildFile *BuildFile) error {
	result, err := d.db.Exec(`
		INSERT INTO build_files (build_id, type, sub_type, size, state, storage_path, upload_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		buildFile.BuildID, buildFile.Type, buildFile.SubType, buildFile.Size,
		buildFile.State, buildFile.StoragePath, buildFile.UploadURL)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	buildFile.ID = id
	return nil
}

func (d *SQLiteDatabase) UpdateBuildFile(buildFile *BuildFile) error {
	_, err := d.db.Exec(`
		UPDATE build_files SET build_id = ?, type = ?, sub_type = ?, size = ?, state = ?, 
		storage_path = ?, upload_url = ?, updated_at = datetime('now')
		WHERE id = ?`,
		buildFile.BuildID, buildFile.Type, buildFile.SubType, buildFile.Size,
		buildFile.State, buildFile.StoragePath, buildFile.UploadURL, buildFile.ID)
	return err
}

// Channel database methods
func (d *SQLiteDatabase) GetChannelByName(name string, uploadID int64) (*Channel, error) {
	channel := &Channel{}
	var currentBuildID sql.NullInt64

	err := d.db.QueryRow(`
		SELECT id, name, upload_id, current_build_id, created_at, updated_at
		FROM channels WHERE name = ? AND upload_id = ?`, name, uploadID).Scan(
		&channel.ID, &channel.Name, &channel.UploadID, &currentBuildID,
		&channel.CreatedAt, &channel.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if currentBuildID.Valid {
		channel.CurrentBuildID = &currentBuildID.Int64
	}

	return channel, nil
}

func (d *SQLiteDatabase) GetChannelsByUploadID(uploadID int64) ([]*Channel, error) {
	rows, err := d.db.Query(`
		SELECT id, name, upload_id, current_build_id, created_at, updated_at
		FROM channels WHERE upload_id = ?`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel := &Channel{}
		var currentBuildID sql.NullInt64

		err := rows.Scan(&channel.ID, &channel.Name, &channel.UploadID, &currentBuildID,
			&channel.CreatedAt, &channel.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if currentBuildID.Valid {
			channel.CurrentBuildID = &currentBuildID.Int64
		}

		channels = append(channels, channel)
	}

	return channels, nil
}

func (d *SQLiteDatabase) CreateChannel(channel *Channel) error {
	var currentBuildID interface{}
	if channel.CurrentBuildID != nil {
		currentBuildID = *channel.CurrentBuildID
	}

	result, err := d.db.Exec(`
		INSERT INTO channels (name, upload_id, current_build_id, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`,
		channel.Name, channel.UploadID, currentBuildID)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	channel.ID = id
	return nil
}

func (d *SQLiteDatabase) UpdateChannel(channel *Channel) error {
	var currentBuildID interface{}
	if channel.CurrentBuildID != nil {
		currentBuildID = *channel.CurrentBuildID
	}

	_, err := d.db.Exec(`
		UPDATE channels SET name = ?, upload_id = ?, current_build_id = ?, updated_at = datetime('now')
		WHERE id = ?`,
		channel.Name, channel.UploadID, currentBuildID, channel.ID)
	return err
}

// UploadSession methods removed - using MinIO presigned URLs instead

// Initialize database with migrations
func (d *SQLiteDatabase) Migrate() error {
	// Read and execute migration
	migrationSQL := `
-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    api_key TEXT UNIQUE NOT NULL,
    role TEXT DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create games table
CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    short_text TEXT,
    type TEXT DEFAULT 'default',
    classification TEXT DEFAULT 'game',
    url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create uploads table
CREATE TABLE IF NOT EXISTS uploads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    filename TEXT NOT NULL,
    display_name TEXT,
    size INTEGER DEFAULT 0,
    storage TEXT DEFAULT 'hosted',
    type TEXT DEFAULT 'default',
    platforms TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id)
);

-- Create builds table
CREATE TABLE IF NOT EXISTS builds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    upload_id INTEGER NOT NULL,
    user_version TEXT,
    parent_build_id INTEGER,
    state TEXT DEFAULT 'started',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (upload_id) REFERENCES uploads(id),
    FOREIGN KEY (parent_build_id) REFERENCES builds(id)
);

-- Create build_files table
CREATE TABLE IF NOT EXISTS build_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    build_id INTEGER NOT NULL,
    type TEXT NOT NULL,
    sub_type TEXT DEFAULT 'default',
    size INTEGER DEFAULT 0,
    state TEXT DEFAULT 'uploading',
    storage_path TEXT,
    upload_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (build_id) REFERENCES builds(id)
);

-- Create channels table
CREATE TABLE IF NOT EXISTS channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    upload_id INTEGER NOT NULL,
    current_build_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (upload_id) REFERENCES uploads(id),
    FOREIGN KEY (current_build_id) REFERENCES builds(id),
    UNIQUE(name, upload_id)
);

-- Create upload_sessions table
CREATE TABLE IF NOT EXISTS upload_sessions (
    id TEXT PRIMARY KEY,
    build_file_id INTEGER NOT NULL,
    storage_path TEXT NOT NULL,
    size INTEGER DEFAULT 0,
    state TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (build_file_id) REFERENCES build_files(id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key);
CREATE INDEX IF NOT EXISTS idx_games_user_id ON games(user_id);
CREATE INDEX IF NOT EXISTS idx_uploads_game_id ON uploads(game_id);
CREATE INDEX IF NOT EXISTS idx_builds_upload_id ON builds(upload_id);
CREATE INDEX IF NOT EXISTS idx_build_files_build_id ON build_files(build_id);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_build_file_id ON upload_sessions(build_file_id);
	`

	_, err := d.db.Exec(migrationSQL)
	return err
}
