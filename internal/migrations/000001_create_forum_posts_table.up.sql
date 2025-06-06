CREATE TABLE forum_posts (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id INT REFERENCES forum_categories(id) ON DELETE SET NULL,
    category_name VARCHAR(50) REFERENCES forum_categories(name) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL, -- Markdown
    is_pinned BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE, -- By admin
    view_count INT DEFAULT 0,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(), -- For sorting by latest activity
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
