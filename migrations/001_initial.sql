-- Create users table
CREATE TABLE users (
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
CREATE TABLE games (
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
CREATE TABLE uploads (
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
CREATE TABLE builds (
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
CREATE TABLE build_files (
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
CREATE TABLE channels (
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
CREATE TABLE upload_sessions (
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
CREATE INDEX idx_users_api_key ON users(api_key);
CREATE INDEX idx_games_user_id ON games(user_id);
CREATE INDEX idx_uploads_game_id ON uploads(game_id);
CREATE INDEX idx_builds_upload_id ON builds(upload_id);
CREATE INDEX idx_build_files_build_id ON build_files(build_id);
CREATE INDEX idx_channels_name ON channels(name);
CREATE INDEX idx_upload_sessions_build_file_id ON upload_sessions(build_file_id);