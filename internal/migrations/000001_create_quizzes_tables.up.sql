-- This single, generic table can store all quizzes, regardless of what they belong to.
CREATE TABLE quizzes (
    id BIGSERIAL PRIMARY KEY,
    -- Polymorphic association to link a quiz to its owner
    owner_type VARCHAR(50), -- e.g., 'course', 'course_chapter', 'course_video'
    owner_id BIGINT,
    title VARCHAR(255) NOT NULL,
    -- The entire quiz structure (questions, options, answers) is stored here.
    questions JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- An index to quickly find all quizzes for a specific owner (e.g., all quizzes in a chapter).
CREATE INDEX idx_quizzes_owner ON quizzes (owner_type, owner_id);
