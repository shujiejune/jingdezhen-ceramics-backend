CREATE TABLE user_note_links (
    id BIGSERIAL PRIMARY KEY,
    user_note_id BIGINT NOT NULL REFERENCES user_notes(id) ON DELETE CASCADE,
    linked_entity_type VARCHAR(50) NOT NULL, -- 'artwork', 'course_chapter', 'course_video_timestamp', 'engage_article_paragraph', 'forum_post'
    linked_entity_id BIGINT,          -- For entities with integer IDs (artwork_id, chapter_id, forum_post_id)
    link_description TEXT,             -- Optional: User's description of why this link is relevant
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_note_id, linked_entity_type, linked_entity_id, -- Ensure a note doesn't link to the same thing twice
);
