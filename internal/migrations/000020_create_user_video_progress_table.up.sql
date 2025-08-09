CREATE TABLE user_video_progress (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_block_id BIGINT NOT NULL REFERENCES chapter_content_blocks(id) ON DELETE CASCADE,
    last_stopped_at BIGINT NOT NULL DEFAULT 0, -- Store seconds as a long integer
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, content_block_id)
);
