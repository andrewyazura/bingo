CREATE TABLE IF NOT EXISTS rooms (
    slug VARCHAR(50) PRIMARY KEY,
    mode INT NOT NULL,
    size INT NOT NULL,
    wordlist JSONB NOT NULL,
    free_space BOOLEAN NOT NULL,
    password_hash VARCHAR(255) NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    player_id VARCHAR(100) PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255),
    session_token VARCHAR(255) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS room_access (
    room_slug VARCHAR(50) REFERENCES rooms(slug) ON DELETE CASCADE,
    player_id VARCHAR(100),
    unlocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_slug, player_id)
);

CREATE TABLE IF NOT EXISTS game_events (
    id BIGSERIAL PRIMARY KEY,
    room_slug VARCHAR(50) NOT NULL REFERENCES rooms(slug) ON DELETE CASCADE,
    player_id VARCHAR(100) NOT NULL,
    event_type INT NOT NULL,
    tile_index INT,
    tile_word VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_game_events_room_slug ON game_events(room_slug);
