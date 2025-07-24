CREATE TABLE user_content_block_progress (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_block_id BIGINT NOT NULL REFERENCES chapter_content_blocks(id) ON DELETE CASCADE,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, content_block_id)
);
