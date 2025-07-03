-- name: GetUser :one
SELECT
    *
FROM
    users
WHERE
    id = sqlc.arg('userId');

-- name: UpdateUser :exec
UPDATE
    users
SET
    name = ?
WHERE
    id = ?;

-- name: GetPostsFromUserId :many
SELECT
    *
FROM
    posts
WHERE
    author_id = sqlc.arg('userId')
    AND subreddit_id = sqlc.arg('subredditId');

-- name: UpdatePost :one
UPDATE
    posts
SET
    content = ?,
    updated_at = ?
WHERE
    id = ?
RETURNING
    *;

-- name: CreateUser :one
INSERT INTO
    users (name)
VALUES
    (?)
RETURNING
    *;

-- name: PostWithUser :one
SELECT
    posts.*,
    sqlc.embed(u) AS author
FROM
    posts
    JOIN users u ON posts.author_id = users.id
WHERE
    users.id = ?;
