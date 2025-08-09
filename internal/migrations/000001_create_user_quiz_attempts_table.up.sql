CREATE TABLE user_quiz_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quiz_id BIGINT NOT NULL, -- Can reference chapter_quizzes.id or video_inserted_quizzes.id
    quiz_type VARCHAR(20) NOT NULL, -- 'chapter_quiz', 'video_quiz'
    attempt_data JSONB, -- User's answers
    score INT,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
