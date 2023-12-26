-- name: GetBookById :one
SELECT
    *
FROM
    books
WHERE
    book_id = sqlc.arg(book_id);

-- name: InsertBook :one
INSERT INTO
    books(title, author, price, description, author_bio)
VALUES
    (
        sqlc.arg(title),
        sqlc.arg(author),
        sqlc.arg(price),
        sqlc.arg(description),
        sqlc.arg(author_bio)
    ) RETURNING book_id;