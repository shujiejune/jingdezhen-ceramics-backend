CREATE TABLE user_enrollments (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_visited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A user can only be enrolled in a course once.
    PRIMARY KEY (user_id, course_id)
);

-- Add an index for efficiently looking up courses by user, ordered by last visit.
CREATE INDEX idx_user_enrollments_last_visited ON user_enrollments(user_id, last_visited_at DESC);
