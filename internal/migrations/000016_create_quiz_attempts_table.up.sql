CREATE TABLE quiz_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quiz_id BIGINT NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    answers JSONB NOT NULL,
    score INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('graded', 'pending_manual_grade')),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
