-- for gallery
CREATE TABLE artworks (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    artist_id INT REFERENCES artists(id) ON DELETE SET NULL, -- Can be null if artist unknown or not in DB
    artist_name_override VARCHAR(255), -- If artist not in artists table or to override
    thumbnail_url TEXT NOT NULL,
    description TEXT,
    period VARCHAR(50),
    dimensions VARCHAR(100), -- e.g., "20cm x 30cm x 15cm"
    category VARCHAR(100), -- e.g., "blue and white"
    utensil VARCHAR(100), -- e.g., "Vase", "Teaware", "Sculpture"
    introduction TEXT, -- 1-2 lines: won prize, creator's intention
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
