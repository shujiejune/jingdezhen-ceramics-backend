CREATE TABLE forum_comment_likes (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id INT NOT NULL REFERENCES forum_comments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, comment_id)
);
