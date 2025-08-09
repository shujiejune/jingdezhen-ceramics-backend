-- for gallery
CREATE TABLE artworks (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    artist_id BIGINT REFERENCES artists(id) ON DELETE SET NULL, -- Can be null if artist unknown or not in DB
    artist_name_override VARCHAR(255), -- If artist not in artists table or to override
    thumbnail_url TEXT NOT NULL,
    description TEXT,
    period VARCHAR(50) NOT NULL,
    dimensions VARCHAR(100), -- e.g., "20cm x 30cm x 15cm"
    category VARCHAR(100) NOT NULL, -- e.g., "blue and white"
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
