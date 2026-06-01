-- Rooms table holds the blueprints (stateless creation)
CREATE TABLE IF NOT EXISTS rooms (
    slug VARCHAR(50) PRIMARY KEY,
    mode INT NOT NULL,           -- 0: Competitive, 1: Collaborative
    size INT NOT NULL,           -- e.g., 5 for 5x5
    wordlist JSONB NOT NULL,     -- Array of words to generate boards
    free_space BOOLEAN NOT NULL,
    password_hash VARCHAR(255) NULL, -- Optional password protection
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    player_id VARCHAR(100) PRIMARY KEY,       -- Maps to the 'guest_xxxx' cookie
    email VARCHAR(255) UNIQUE,                -- Nullable until registered
    password_hash VARCHAR(255),               -- Nullable until registered
    session_token VARCHAR(255) UNIQUE,        -- Cryptographically secure token for authenticated sessions
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS room_access (
    room_slug VARCHAR(50) REFERENCES rooms(slug) ON DELETE CASCADE,
    player_id VARCHAR(100), -- Maps to the guest or registered player_id
    unlocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_slug, player_id)
);

-- Game Events table holds the entire history of actions for crash recovery
CREATE TABLE IF NOT EXISTS game_events (
    id BIGSERIAL PRIMARY KEY,
    room_slug VARCHAR(50) NOT NULL REFERENCES rooms(slug) ON DELETE CASCADE,
    player_id VARCHAR(100) NOT NULL,
    event_type INT NOT NULL,     -- 0: Bingo, 1: OneToBingo, 2: TileMarked, 3: TileUnmarked
    tile_index INT,              -- Can be NULL for Bingo/Joined events
    tile_word VARCHAR(255),      -- The actual word that was marked (for analytics)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for extremely fast Actor event replays
CREATE INDEX IF NOT EXISTS idx_game_events_room_slug ON game_events(room_slug);
