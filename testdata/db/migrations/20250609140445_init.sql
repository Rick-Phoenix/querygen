-- migrate:up
CREATE TABLE users (
    id integer PRIMARY KEY,
    name text NOT NULL UNIQUE,
    created_at datetime NOT NULL DEFAULT (strftime ('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE subreddits (
    id integer PRIMARY KEY,
    name text NOT NULL UNIQUE,
    description text,
    created_at datetime NOT NULL DEFAULT (strftime ('%Y-%m-%dT%H:%M:%fZ', 'now')),
    creator_id integer,
    FOREIGN KEY (creator_id) REFERENCES users (id) ON DELETE
    SET
        NULL
);

CREATE TABLE posts (
    id integer PRIMARY KEY,
    title text NOT NULL,
    content text,
    created_at datetime NOT NULL DEFAULT (strftime ('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at datetime NOT NULL DEFAULT (strftime ('%Y-%m-%dT%H:%M:%fZ', 'now')),
    author_id integer NOT NULL,
    subreddit_id integer NOT NULL,
    FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (subreddit_id) REFERENCES subreddits (id) ON DELETE CASCADE
);

CREATE TABLE comments (
    id integer PRIMARY KEY,
    text_content text NOT NULL,
    created_at datetime NOT NULL DEFAULT (strftime ('%Y-%m-%dT%H:%M:%fZ', 'now')),
    author_id integer NOT NULL,
    post_id integer NOT NULL,
    parent_comment_id integer,
    FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    FOREIGN KEY (parent_comment_id) REFERENCES comments (id) ON DELETE CASCADE
);

CREATE TABLE user_subscriptions (
    user_id integer NOT NULL,
    subreddit_id integer NOT NULL,
    created_at datetime NOT NULL DEFAULT (strftime ('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (user_id, subreddit_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (subreddit_id) REFERENCES subreddits (id) ON DELETE CASCADE
);

-- migrate:down
DROP TABLE IF EXISTS user_subscriptions;

DROP TABLE IF EXISTS comments;

DROP TABLE IF EXISTS posts;

DROP TABLE IF EXISTS subreddits;

DROP TABLE IF EXISTS users;
