package models

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// PostgresDatabase implements the Database interface using PostgreSQL
type PostgresDatabase struct {
	db *sql.DB
}

// NewPostgresDatabase creates a new PostgreSQL database connection
func NewPostgresDatabase() (*PostgresDatabase, error) {
	host := getEnvOrDefault("POSTGRES_HOST", "localhost")
	port := getEnvOrDefault("POSTGRES_PORT", "5432")
	user := getEnvOrDefault("POSTGRES_USER", "postgres")
	password := getEnvOrDefault("POSTGRES_PASSWORD", "postgres")
	dbname := getEnvOrDefault("POSTGRES_DB", "butler")
	sslmode := getEnvOrDefault("POSTGRES_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	pgDB := &PostgresDatabase{db: db}

	// Run migrations
	if err := pgDB.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	return pgDB, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// migrate runs the database migrations
func (d *PostgresDatabase) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			display_name VARCHAR(255) NOT NULL,
			api_key VARCHAR(255) UNIQUE NOT NULL,
			role VARCHAR(50) DEFAULT 'user' CHECK (role IN ('user', 'admin')),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS games (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title VARCHAR(255) NOT NULL,
			short_text TEXT,
			type VARCHAR(50) DEFAULT 'default',
			classification VARCHAR(50) DEFAULT 'game',
			url VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS uploads (
			id SERIAL PRIMARY KEY,
			game_id INTEGER REFERENCES games(id),
			filename VARCHAR(255),
			display_name VARCHAR(255),
			storage VARCHAR(255),
			size BIGINT DEFAULT 0,
			type VARCHAR(50),
			platforms TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id SERIAL PRIMARY KEY,
			upload_id INTEGER REFERENCES uploads(id),
			name VARCHAR(255) NOT NULL,
			build_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS builds (
			id SERIAL PRIMARY KEY,
			upload_id INTEGER REFERENCES uploads(id),
			parent_build_id INTEGER REFERENCES builds(id),
			user_version VARCHAR(255),
			state VARCHAR(50) DEFAULT 'started',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS build_files (
			id SERIAL PRIMARY KEY,
			build_id INTEGER REFERENCES builds(id),
			type VARCHAR(50) NOT NULL,
			sub_type VARCHAR(50) NOT NULL,
			state VARCHAR(50) DEFAULT 'uploading',
			storage_path VARCHAR(255),
			size BIGINT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS upload_sessions (
			id VARCHAR(255) PRIMARY KEY,
			build_file_id INTEGER REFERENCES build_files(id),
			storage_path VARCHAR(255),
			size BIGINT DEFAULT 0,
			state VARCHAR(50) DEFAULT 'uploading',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			return fmt.Errorf("failed to execute migration: %v", err)
		}
	}

	return nil
}

// Close closes the database connection
func (d *PostgresDatabase) Close() error {
	return d.db.Close()
}

// User methods
func (d *PostgresDatabase) GetUserByAPIKey(apiKey string) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, username, display_name, api_key, role, is_active, created_at, updated_at 
		FROM users WHERE api_key = $1 AND is_active = true`, apiKey).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *PostgresDatabase) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, username, display_name, api_key, role, is_active, created_at, updated_at 
		FROM users WHERE id = $1`, id).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *PostgresDatabase) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, username, display_name, api_key, role, is_active, created_at, updated_at 
		FROM users WHERE username = $1`, username).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *PostgresDatabase) CreateUser(user *User) error {
	// Set default values if not provided
	if user.Role == "" {
		user.Role = "user"
	}
	if !user.IsActive {
		user.IsActive = true
	}

	err := d.db.QueryRow(`
		INSERT INTO users (username, display_name, api_key, role, is_active)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		user.Username, user.DisplayName, user.APIKey, user.Role, user.IsActive).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt)
	return err
}

func (d *PostgresDatabase) UpdateUser(user *User) error {
	_, err := d.db.Exec(`
		UPDATE users SET username = $1, display_name = $2, api_key = $3, role = $4, is_active = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6`,
		user.Username, user.DisplayName, user.APIKey, user.Role, user.IsActive, user.ID)
	return err
}

func (d *PostgresDatabase) ListUsers() ([]*User, error) {
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

// Game methods
func (d *PostgresDatabase) GetGameByID(id int64) (*User, *Game, error) {
	game := &Game{}
	user := &User{}

	err := d.db.QueryRow(`
		SELECT
			g.id, g.user_id, g.title, g.short_text, g.type, g.classification, g.url, g.created_at, g.updated_at,
			u.id, u.username, u.display_name, u.api_key, u.role, u.is_active, u.created_at, u.updated_at
		FROM games g
		JOIN users u ON g.user_id = u.id
		WHERE g.id = $1`, id).Scan(
		&game.ID, &game.UserID, &game.Title, &game.ShortText, &game.Type, &game.Classification, &game.URL, &game.CreatedAt, &game.UpdatedAt,
		&user.ID, &user.Username, &user.DisplayName, &user.APIKey, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, nil, err
	}

	return user, game, nil
}

func (d *PostgresDatabase) GetGameByUserAndTitle(userID int64, title string) (*Game, error) {
	game := &Game{}
	err := d.db.QueryRow(`
		SELECT id, user_id, title, short_text, type, classification, url, created_at, updated_at
		FROM games WHERE user_id = $1 AND title = $2`, userID, title).Scan(
		&game.ID, &game.UserID, &game.Title, &game.ShortText, &game.Type, &game.Classification, &game.URL, &game.CreatedAt, &game.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return game, nil
}

func (d *PostgresDatabase) CreateGame(game *Game) error {
	err := d.db.QueryRow(`
		INSERT INTO games (user_id, title, short_text, type, classification, url)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		game.UserID, game.Title, game.ShortText, game.Type, game.Classification, game.URL).Scan(
		&game.ID, &game.CreatedAt, &game.UpdatedAt)
	return err
}

// Upload methods
func (d *PostgresDatabase) GetUploadsByGameID(gameID int64) ([]*Upload, error) {
	rows, err := d.db.Query(`
		SELECT id, game_id, filename, display_name, storage, size, created_at, updated_at
		FROM uploads WHERE game_id = $1`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uploads []*Upload
	for rows.Next() {
		upload := &Upload{}
		err := rows.Scan(&upload.ID, &upload.GameID, &upload.Filename, &upload.DisplayName,
			&upload.Storage, &upload.Size, &upload.CreatedAt, &upload.UpdatedAt)
		if err != nil {
			return nil, err
		}
		uploads = append(uploads, upload)
	}
	return uploads, nil
}

func (d *PostgresDatabase) CreateUpload(upload *Upload) error {
	err := d.db.QueryRow(`
		INSERT INTO uploads (game_id, filename, display_name, storage, size)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		upload.GameID, upload.Filename, upload.DisplayName, upload.Storage, upload.Size).Scan(
		&upload.ID, &upload.CreatedAt, &upload.UpdatedAt)
	return err
}

func (d *PostgresDatabase) GetUploadByID(id int64) (*Upload, error) {
	upload := &Upload{}
	err := d.db.QueryRow(`
		SELECT id, game_id, filename, display_name, storage, size, type, platforms, created_at, updated_at
		FROM uploads WHERE id = $1`, id).Scan(
		&upload.ID, &upload.GameID, &upload.Filename, &upload.DisplayName,
		&upload.Storage, &upload.Size, &upload.Type, &upload.Platforms, &upload.CreatedAt, &upload.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return upload, nil
}

func (d *PostgresDatabase) GetGamesByUserID(userID int64) ([]*Game, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, title, short_text, type, classification, url, created_at, updated_at
		FROM games WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*Game
	for rows.Next() {
		game := &Game{}
		err := rows.Scan(&game.ID, &game.UserID, &game.Title, &game.ShortText, &game.Type,
			&game.Classification, &game.URL, &game.CreatedAt, &game.UpdatedAt)
		if err != nil {
			return nil, err
		}
		games = append(games, game)
	}
	return games, nil
}

// Build methods
func (d *PostgresDatabase) GetBuildByID(id int64) (*Build, error) {
	build := &Build{}
	err := d.db.QueryRow(`
		SELECT id, upload_id, parent_build_id, user_version, state, created_at, updated_at
		FROM builds WHERE id = $1`, id).Scan(
		&build.ID, &build.UploadID, &build.ParentBuildID, &build.UserVersion,
		&build.State, &build.CreatedAt, &build.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return build, nil
}

func (d *PostgresDatabase) CreateBuild(build *Build) error {
	err := d.db.QueryRow(`
		INSERT INTO builds (upload_id, parent_build_id, user_version, state)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		build.UploadID, build.ParentBuildID, build.UserVersion, build.State).Scan(
		&build.ID, &build.CreatedAt, &build.UpdatedAt)
	return err
}

func (d *PostgresDatabase) UpdateBuild(build *Build) error {
	_, err := d.db.Exec(`
		UPDATE builds SET upload_id = $1, parent_build_id = $2, user_version = $3, state = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5`,
		build.UploadID, build.ParentBuildID, build.UserVersion, build.State, build.ID)
	return err
}

func (d *PostgresDatabase) GetBuildsByUploadID(uploadID int64) ([]*Build, error) {
	rows, err := d.db.Query(`
		SELECT id, upload_id, parent_build_id, user_version, state, created_at, updated_at
		FROM builds WHERE upload_id = $1`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var builds []*Build
	for rows.Next() {
		build := &Build{}
		err := rows.Scan(&build.ID, &build.UploadID, &build.ParentBuildID, &build.UserVersion,
			&build.State, &build.CreatedAt, &build.UpdatedAt)
		if err != nil {
			return nil, err
		}
		builds = append(builds, build)
	}
	return builds, nil
}

// BuildFile methods
func (d *PostgresDatabase) GetBuildFilesByBuildID(buildID int64) ([]*BuildFile, error) {
	rows, err := d.db.Query(`
		SELECT id, build_id, type, sub_type, state, storage_path, size, created_at, updated_at
		FROM build_files WHERE build_id = $1`, buildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*BuildFile
	for rows.Next() {
		file := &BuildFile{}
		err := rows.Scan(&file.ID, &file.BuildID, &file.Type, &file.SubType,
			&file.State, &file.StoragePath, &file.Size, &file.CreatedAt, &file.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (d *PostgresDatabase) GetBuildFileByID(id int64) (*BuildFile, error) {
	file := &BuildFile{}
	err := d.db.QueryRow(`
		SELECT id, build_id, type, sub_type, state, storage_path, size, created_at, updated_at
		FROM build_files WHERE id = $1`, id).Scan(
		&file.ID, &file.BuildID, &file.Type, &file.SubType,
		&file.State, &file.StoragePath, &file.Size, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (d *PostgresDatabase) CreateBuildFile(file *BuildFile) error {
	err := d.db.QueryRow(`
		INSERT INTO build_files (build_id, type, sub_type, state, storage_path, size)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		file.BuildID, file.Type, file.SubType, file.State, file.StoragePath, file.Size).Scan(
		&file.ID, &file.CreatedAt, &file.UpdatedAt)
	return err
}

func (d *PostgresDatabase) UpdateBuildFile(file *BuildFile) error {
	_, err := d.db.Exec(`
		UPDATE build_files SET build_id = $1, type = $2, sub_type = $3, state = $4, storage_path = $5, size = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7`,
		file.BuildID, file.Type, file.SubType, file.State, file.StoragePath, file.Size, file.ID)
	return err
}

// Channel methods
func (d *PostgresDatabase) GetChannelsByUploadID(uploadID int64) ([]*Channel, error) {
	rows, err := d.db.Query(`
		SELECT id, upload_id, name, build_id, created_at, updated_at
		FROM channels WHERE upload_id = $1`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel := &Channel{}
		var buildID sql.NullInt64
		err := rows.Scan(&channel.ID, &channel.UploadID, &channel.Name, &buildID,
			&channel.CreatedAt, &channel.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if buildID.Valid {
			channel.CurrentBuildID = &buildID.Int64
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

func (d *PostgresDatabase) GetChannelByName(name string, uploadID int64) (*Channel, error) {
	channel := &Channel{}
	var buildID sql.NullInt64
	err := d.db.QueryRow(`
		SELECT id, upload_id, name, build_id, created_at, updated_at
		FROM channels WHERE name = $1 AND upload_id = $2`, name, uploadID).Scan(
		&channel.ID, &channel.UploadID, &channel.Name, &buildID,
		&channel.CreatedAt, &channel.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if buildID.Valid {
		channel.CurrentBuildID = &buildID.Int64
	}
	return channel, nil
}

func (d *PostgresDatabase) CreateChannel(channel *Channel) error {
	err := d.db.QueryRow(`
		INSERT INTO channels (upload_id, name, build_id)
		VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		channel.UploadID, channel.Name, channel.CurrentBuildID).Scan(
		&channel.ID, &channel.CreatedAt, &channel.UpdatedAt)
	return err
}

func (d *PostgresDatabase) UpdateChannel(channel *Channel) error {
	_, err := d.db.Exec(`
		UPDATE channels SET upload_id = $1, name = $2, build_id = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4`,
		channel.UploadID, channel.Name, channel.CurrentBuildID, channel.ID)
	return err
}
