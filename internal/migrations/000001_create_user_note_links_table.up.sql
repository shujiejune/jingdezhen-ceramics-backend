CREATE TABLE user_note_links (
    id SERIAL PRIMARY KEY,
    user_note_id INT NOT NULL REFERENCES user_notes(id) ON DELETE CASCADE,
    linked_entity_type VARCHAR(50) NOT NULL, -- 'artwork', 'course_chapter', 'course_video_timestamp', 'engage_article_paragraph', 'forum_post'
    linked_entity_id_int INT,          -- For entities with integer IDs (artwork_id, chapter_id, forum_post_id)
    linked_entity_id_uuid UUID,        -- For entities with UUIDs (if any)
    linked_entity_id_string VARCHAR(255), -- For other string-based IDs (e.g., paragraph ID within an article)
    link_description TEXT,             -- Optional: User's description of why this link is relevant
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_note_id, linked_entity_type, linked_entity_id_int, linked_entity_id_uuid, linked_entity_id_string), -- Ensure a note doesn't link to the same thing twice
    CHECK (
        (linked_entity_id_int IS NOT NULL)::int +
        (linked_entity_id_uuid IS NOT NULL)::int +
        (linked_entity_id_string IS NOT NULL)::int = 1
    ) -- Add CHECK constraint to ensure only one of linked_entity_id_* is non-null based on type
);
