-- made for PostgreSQL. probably won't work on mysql/sqlite!
BEGIN;
CREATE EXTENSION pgcrypto;
CREATE TABLE movies(
    id SERIAL PRIMARY KEY,
    author_userid INT NOT NULL,
    author_fsid VARCHAR(16) NOT NULL,
    author_name TEXT NOT NULL,
    author_filename VARCHAR(24) UNIQUE NOT NULL,
    uploaded TIMESTAMPTZ DEFAULT now(),
    views INT DEFAULT 0,
    downloads INT DEFAULT 0,
    lock BOOL DEFAULT FALSE,
    deleted BOOL DEFAULT FALSE,
    channelid INT DEFAULT 0
);
CREATE TABLE auth_whitelist(
    id SERIAL PRIMARY KEY,
    fsid VARCHAR(16) NOT NULL
);
CREATE TABLE bans(
    id SERIAL PRIMARY KEY,
    issuer TEXT NOT NULL,
    issued TIMESTAMPTZ DEFAULT now(),
    expires TIMESTAMPTZ DEFAULT now() + interval '24 hours',
    message TEXT DEFAULT 'begone',
    pardon BOOL DEFAULT FALSE,
    affected TEXT NOT NULL
);
CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password VARCHAR(60) NOT NULL,
    admin BOOL DEFAULT FALSE,
    fsid VARCHAR(16) DEFAULT '0000000000000000',
    last_login_ip TEXT DEFAULT 'never logged in before'
);
CREATE TABLE apitokens(
    id SERIAL PRIMARY KEY,
    secret VARCHAR(60) UNIQUE NOT NULL,
    expires TIMESTAMPTZ DEFAULT now() + interval '30 days',
    userid INTEGER NOT NULL
);
CREATE TABLE user_star(
    userid INT NOT NULL,
    movieid INT NOT NULL,
    ys INT DEFAULT 0,
    gs INT DEFAULT 0,
    rs INT DEFAULT 0,
    bs INT DEFAULT 0,
    ps INT DEFAULT 0
);
CREATE TABLE comments(
    id SERIAL PRIMARY KEY,
    userid INT NOT NULL,
    movieid INT NOT NULL,
    is_memo BOOL DEFAULT TRUE,
    content TEXT DEFAULT 'hhhhh',
    posted TIMESTAMPTZ DEFAULT now()
);
CREATE TABLE channels(
    id SERIAL PRIMARY KEY,
    desc_s TEXT DEFAULT 'New channel',
    desc_l TEXT DEFAULT 'Call Luigi?'
)
CREATE FUNCTION get_movie_stars(memoid INT) RETURNS TABLE (yst BIGINT, gst BIGINT, rst BIGINT, bst BIGINT, pst BIGINT) AS $BODY$
BEGIN
    RETURN QUERY SELECT coalesce(sum(ys), 0), coalesce(sum(gs), 0), coalesce(sum(rs), 0), coalesce(sum(bs), 0), coalesce(sum(ps), 0) FROM user_star WHERE user_star.movieid = $1;
END;
$BODY$ STABLE LANGUAGE plpgsql;
CREATE VIEW count_all_movies AS SELECT count(1) FROM movies WHERE deleted = false;
COMMIT;