CREATE TABLE user_chapter_progress (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chapter_id INT NOT NULL REFERENCES course_chapters(id) ON DELETE CASCADE,
    progress_percentage INT NOT NULL DEFAULT 0 CHECK (progress_percentage >= 0 AND progress_percentage <= 100),
    video_last_stopped_at INT DEFAULT 0, -- seconds
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, chapter_id)
);
