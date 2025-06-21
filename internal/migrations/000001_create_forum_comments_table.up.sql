CREATE TABLE forum_comments (
    id SERIAL PRIMARY KEY,
    post_id INT NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id INT REFERENCES forum_comments(id) ON DELETE CASCADE, -- For threaded comments
    content TEXT NOT NULL, -- Markdown
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
