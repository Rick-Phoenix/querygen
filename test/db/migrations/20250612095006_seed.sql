-- migrate:up
INSERT INTO
    users (name)
VALUES
    ("gianfranco");

INSERT INTO
    subreddits (name, creator_id)
VALUES
    ("r/cats", 1);

INSERT INTO
    posts (title, author_id, subreddit_id)
VALUES
    ("cats are neat eh?", 1, 1);

-- migrate:down
DELETE FROM
    users
WHERE
    name = "gianfranco";

DELETE FROM
    subreddits
WHERE
    id = 1;

DELETE FROM
    posts
WHERE
    id = 1;
