CREATE TABLE portfolio_work_images (
    id BIGINT PRIMARY KEY,
    portfolio_work_id BIGINT NOT NULL REFERENCES portfolio_works(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    is_thumbnail BOOLEAN DEFAULT FALSE,
    caption TEXT,
    display_order INT DEFAULT 0
);
