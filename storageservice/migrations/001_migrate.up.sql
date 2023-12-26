CREATE TABLE IF NOT EXISTS books (
    book_id SERIAL PRIMARY KEY NOT NULL,
    title VARCHAR(100) NOT NULL,
    author VARCHAR(100)NOT NULL,
    price FLOAT,
    description VARCHAR(3001),
    author_bio VARCHAR(3000)
);