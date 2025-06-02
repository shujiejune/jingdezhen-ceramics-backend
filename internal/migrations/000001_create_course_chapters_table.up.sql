CREATE TABLE course_chapters (
    id SERIAL PRIMARY KEY,
    course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    display_order INT NOT NULL,
    background_color VARCHAR(7), -- e.g., "#RRGGBB"
    video_url TEXT,
    video_duration INT, -- in seconds
    -- Passages (text and pictures) can be part of chapter content or a separate related table if very complex
    content TEXT, -- For passages, can be Markdown
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, display_order)
);
