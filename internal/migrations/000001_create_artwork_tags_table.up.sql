CREATE TABLE artwork_tags (
    artwork_id INT NOT NULL REFERENCES artworks(id) ON DELETE CASCADE,
    tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (artwork_id, tag_id)
);
