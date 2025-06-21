-- for user profile
CREATE TABLE badges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    icon_url TEXT,
    criteria JSONB -- { type: "learn_streak", value: 7 } or { type: "notes_taken", value: 1 }
);
