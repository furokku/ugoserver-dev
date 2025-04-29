-- made for PostgreSQL. won't work on mysql/sqlite!
BEGIN;
CREATE EXTENSION pgcrypto;
CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    password VARCHAR(60) NOT NULL,
    admin BOOL DEFAULT FALSE,
    fsid VARCHAR(16) DEFAULT '0000000000000000',
    last_login_ip TEXT DEFAULT 'never logged in before',
    expendable_stars INT ARRAY[5] DEFAULT '{0, 0, 0, 0, 0}'-- index 1 should be unused here
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
    userid INT NOT NULL REFERENCES users(id)
);
CREATE TABLE user_star(
    userid INT NOT NULL REFERENCES users(id),
    movieid INT NOT NULL REFERENCES movies(id),
    stars INT ARRAY[5] DEFAULT '{0, 0, 0, 0, 0}' -- 1:yellow,2:green,3:red,4:blue,5:purple
);

-- add stars to a movie and remove them from the user
CREATE FUNCTION update_movie_stars(userid INT, movieid INT, color INT, count INT) RETURNS void AS $$
DECLARE
    available INT;
BEGIN
    IF NOT EXISTS(SELECT 1 FROM user_star WHERE user_star.userid = update_movie_stars.userid AND user_star.movieid = update_movie_stars.movieid) THEN
        INSERT INTO user_star (userid, movieid) VALUES (userid, movieid);
    END IF;

    IF color > 5 OR color < 1 THEN
        RAISE EXCEPTION 'invalid color';
    END IF;
    
    -- get how many stars the user has
    SELECT expendable_stars[color] INTO available FROM users WHERE users.id = update_movie_stars.userid;

    IF count > available AND color <> 1 THEN
        RAISE EXCEPTION 'tried to add more stars than available';
    END IF;
    
    -- remove stars from user (except yellow)
    IF color <> 1 THEN
        UPDATE users SET expendable_stars[color] = available - count WHERE users.id = update_movie_stars.userid;
    END IF;
    
    IF NOT EXISTS(SELECT 1 FROM user_star WHERE user_star.userid = update_movie_stars.userid AND user_star.movieid = update_movie_stars.movieid) THEN
        INSERT INTO user_star (userid, movieid) VALUES (userid, movieid);
    END IF;

    -- add stars to movie
    UPDATE user_star SET stars[color] = stars[color] + count WHERE user_star.userid = update_movie_stars.userid AND user_star.movieid = update_movie_stars.movieid;

END;
$$ LANGUAGE plpgsql;

-- sum of individual star types from users
CREATE FUNCTION get_movie_stars(movieid INT, OUT yst INT, OUT gst INT, OUT rst INT, OUT bst INT, OUT pst INT) AS $$
BEGIN
    SELECT coalesce(sum(stars[1]), 0), coalesce(sum(stars[2]), 0), coalesce(sum(stars[3]), 0), coalesce(sum(stars[4]), 0), coalesce(sum(stars[5]), 0) INTO yst, gst, rst, bst, pst FROM user_star WHERE user_star.movieid = get_movie_stars.movieid;
END;
$$ STABLE LANGUAGE plpgsql;

-- if posted more than 5 movies in the last 30 minutes -> limit
CREATE FUNCTION get_user_ratelimit(userid INT, OUT until TIMESTAMPTZ) AS $$
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