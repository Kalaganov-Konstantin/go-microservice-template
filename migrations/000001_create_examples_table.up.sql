CREATE TABLE IF NOT EXISTS examples (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_examples_email ON examples(email);
CREATE INDEX IF NOT EXISTS idx_examples_created_at ON examples(created_at);
