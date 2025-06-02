CREATE TABLE chapter_quizzes ( -- Quizzes within a chapter (not inline video quizzes)
    id SERIAL PRIMARY KEY,
    chapter_id INT NOT NULL REFERENCES course_chapters(id) ON DELETE CASCADE,
    title VARCHAR(255),
    quiz_data JSONB NOT NULL, -- { questions: [{text: "", options: [], answer: ""}] }
    display_order INT
);
