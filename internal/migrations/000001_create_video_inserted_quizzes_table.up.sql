CREATE TABLE video_inserted_quizzes ( -- Quizzes inserted along the video timeline
    id SERIAL PRIMARY KEY,
    chapter_id INT NOT NULL REFERENCES course_chapters(id) ON DELETE CASCADE, -- Or a direct video_id if videos are more independent
    timestamp_seconds INT NOT NULL, -- When the quiz should appear in the video
    quiz_data JSONB NOT NULL, -- Polls or simple Q&A
    display_order INT
);
