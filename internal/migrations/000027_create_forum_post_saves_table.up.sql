CREATE TABLE forum_post_saves (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id BIGINT NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, post_id)
);
-- Create an index for faster lookups of a user's saved posts.
CREATE INDEX idx_forum_post_saves_user_id ON forum_post_saves(user_id);
