CREATE TABLE forum_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    display_order INT DEFAULT 0
);
