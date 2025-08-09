CREATE TABLE chapter_content_blocks (
    id BIGSERIAL PRIMARY KEY,
    chapter_id BIGINT NOT NULL REFERENCES course_chapters(id) ON DELETE CASCADE,
    -- Use a CHECK constraint for data integrity
    course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('video', 'reading', 'assignment', 'quiz')),
    content JSONB NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- A chapter cannot have two blocks with the same display order
    UNIQUE (course_id, chapter_id, display_order)
);

-- Index for quickly fetching all blocks for a chapter
CREATE INDEX idx_content_blocks_chapter_id ON chapter_content_blocks(chapter_id);
