CREATE TABLE artwork_images (
    id SERIAL PRIMARY KEY,
    artwork_id INT NOT NULL REFERENCES artworks(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    caption TEXT,
    display_order INT DEFAULT 0
);
