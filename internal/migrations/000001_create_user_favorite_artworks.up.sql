CREATE TABLE user_favorite_artworks (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artwork_id INT NOT NULL REFERENCES artworks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, artwork_id)
);
