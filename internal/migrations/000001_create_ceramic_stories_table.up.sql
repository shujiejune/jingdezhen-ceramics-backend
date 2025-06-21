CREATE TABLE ceramic_stories (
    id SERIAL PRIMARY KEY,
    dynasty_name VARCHAR(100) NOT NULL,
    period VARCHAR(50), -- e.g., "Early Ming", "Late Qing"
    start_year INT,
    end_year INT,
    description TEXT NOT NULL,
    characteristics_craft TEXT,
    characteristics_art TEXT,
    image_url TEXT,
    takeaways TEXT, -- Brief key points
    display_order INT UNIQUE -- For timeline ordering
);
