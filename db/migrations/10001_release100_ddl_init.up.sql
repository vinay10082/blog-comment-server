CREATE TABLE IF NOT EXISTS sample_comment (
    id SERIAL PRIMARY KEY,
    post_id INT NOT NULL,
    content TEXT
);
