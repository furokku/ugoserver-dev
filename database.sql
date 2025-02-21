-- made for PostgreSQL. won't work on mysql/sqlite!
BEGIN;
CREATE EXTENSION pgcrypto;
CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    password VARCHAR(60) NOT NULL,
    admin BOOL DEFAULT FALSE,
    fsid VARCHAR(16) DEFAULT '0000000000000000',
    last_login_ip TEXT DEFAULT 'never logged in before'
);
CREATE TABLE channels(
    id SERIAL PRIMARY KEY,
    desc_s TEXT DEFAULT 'New channel',
    desc_l TEXT DEFAULT 'Call Luigi?'
);
CREATE TABLE movies(
    id SERIAL PRIMARY KEY,
    author_userid INT NOT NULL REFERENCES users(id),
    author_fsid VARCHAR(16) NOT NULL,
    author_name TEXT NOT NULL,
    author_filename VARCHAR(24) NOT NULL,
    posted TIMESTAMPTZ DEFAULT now(),
    views INT DEFAULT 0,
    downloads INT DEFAULT 0,
    lock BOOL DEFAULT FALSE,
    deleted BOOL DEFAULT FALSE,
    channelid INT NOT NULL REFERENCES channels(id)
);
CREATE TABLE comments(
    id SERIAL PRIMARY KEY,
    userid INT NOT NULL REFERENCES users(id),
    movieid INT NOT NULL REFERENCES movies(id),
    is_memo BOOL DEFAULT TRUE,
    content TEXT DEFAULT 'hhhhh',
    posted TIMESTAMPTZ DEFAULT now()
);
CREATE TABLE auth_whitelist(
    id SERIAL PRIMARY KEY,
    userfsid VARCHAR(16)
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
CREATE TABLE apitokens(
    id SERIAL PRIMARY KEY,
    secret VARCHAR(60) UNIQUE NOT NULL,
    expires TIMESTAMPTZ DEFAULT now() + interval '30 days',
    userid INTEGER NOT NULL REFERENCES users(id)
);
CREATE TABLE user_star(
    userid INT NOT NULL REFERENCES users(id),
    movieid INT NOT NULL REFERENCES movies(id),
    ys INT DEFAULT 0,
    gs INT DEFAULT 0,
    rs INT DEFAULT 0,
    bs INT DEFAULT 0,
    ps INT DEFAULT 0
);

CREATE FUNCTION get_movie_stars(movieid INT) RETURNS TABLE (yst BIGINT, gst BIGINT, rst BIGINT, bst BIGINT, pst BIGINT) AS $$
BEGIN
    RETURN QUERY SELECT coalesce(sum(ys), 0), coalesce(sum(gs), 0), coalesce(sum(rs), 0), coalesce(sum(bs), 0), coalesce(sum(ps), 0) FROM user_star WHERE user_star.movieid = get_movie_stars.movieid;
END;
$$ STABLE LANGUAGE plpgsql;

CREATE FUNCTION get_user_movie_ratelimit(userid INT, OUT until TIMESTAMPTZ) AS $$
DECLARE
    latest TIMESTAMPTZ;
    total INT = 0;
BEGIN
    WITH l AS (
        SELECT posted, count(*) OVER (), row_number() OVER (ORDER BY posted DESC) AS rn FROM movies WHERE author_userid = userid AND posted > now() - interval '30 minutes'
    )
    SELECT posted, count INTO latest, total FROM l WHERE rn = count;

    IF total >= 5 THEN
        until := latest + interval '30 minutes';
    ELSE
        until := NULL;
    END IF;
END;
$$ STABLE LANGUAGE plpgsql;
COMMIT;