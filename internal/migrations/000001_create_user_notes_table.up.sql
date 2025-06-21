-- for gallery artworks, course chapters, user profile entry
CREATE TABLE user_notes (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    content TEXT NOT NULL, -- Markdown content
    entity_type VARCHAR(50) NOT NULL, -- 'artwork', 'course_chapter'
    entity_id INT NOT NULL,
    is_published_to_forum BOOLEAN DEFAULT FALSE,
    forum_post_id INT REFERENCES forum_posts(id) ON DELETE SET NULL, -- If published
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, entity_type, entity_id) -- A user can have one note per entity
);
