-- made for PostgreSQL. won't work on mysql/sqlite!
BEGIN;
CREATE EXTENSION pgcrypto;
CREATE TYPE jump_resource AS ENUM ('movie', 'channel', 'user');
CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL, -- get this from the ds
    password VARCHAR(60) NOT NULL,
    admin BOOL DEFAULT FALSE,
    fsid VARCHAR(16) DEFAULT '0000000000000000',
    last_login_time TIMESTAMPTZ DEFAULT now(),
    last_login_ip TEXT DEFAULT 'never logged in before',
    expendable_stars INT ARRAY[5] DEFAULT '{0, 0, 0, 0, 0}',-- first index should be unused here
    deleted BOOL DEFAULT FALSE
);
CREATE TABLE channels(
    id SERIAL PRIMARY KEY,

    chname TEXT DEFAULT 'New channel',
    dsc TEXT DEFAULT 'Call Luigi?',

    deleted BOOL DEFAULT FALSE
);
CREATE TABLE movies(
    id SERIAL PRIMARY KEY,
    channelid INT NOT NULL REFERENCES channels(id),

    author_userid INT NOT NULL REFERENCES users(id),
    author_fsid VARCHAR(16) NOT NULL,
    author_name TEXT NOT NULL,
    author_filename VARCHAR(24) NOT NULL,
    og_author_fsid VARCHAR(16) NOT NULL,
    og_author_name TEXT NOT NULL,
    og_author_filename_fragment VARCHAR(17) NOT NULL,

    views INT DEFAULT 0,
    downloads INT DEFAULT 0,

    lock BOOL DEFAULT FALSE,
    deleted BOOL DEFAULT FALSE,
    posted TIMESTAMPTZ DEFAULT now(),
    last_modified TIMESTAMPTZ NOT NULL
);
CREATE TABLE comments(
    id SERIAL PRIMARY KEY,

    userid INT NOT NULL REFERENCES users(id),
    movieid INT NOT NULL REFERENCES movies(id),

    is_memo BOOL DEFAULT TRUE,
    content TEXT DEFAULT 'hhhhh',

    posted TIMESTAMPTZ DEFAULT now(),
    deleted BOOL DEFAULT FALSE
);
CREATE TABLE auth_whitelist(
    id SERIAL PRIMARY KEY,
    fsid VARCHAR(16)
);
CREATE TABLE bans(
    id SERIAL PRIMARY KEY,

    issuer TEXT NOT NULL,
    message TEXT DEFAULT 'begone',

    issued TIMESTAMPTZ DEFAULT now(),
    expires TIMESTAMPTZ DEFAULT now() + interval '24 hours',

    pardon BOOL DEFAULT FALSE,
    affected TEXT NOT NULL -- ip or fsid
);
CREATE TABLE apitokens(
    id SERIAL PRIMARY KEY,
    userid INT NOT NULL REFERENCES users(id),

    secret VARCHAR(60) UNIQUE NOT NULL,

    created TIMESTAMPTZ DEFAULT now()
);
CREATE TABLE user_star(
    userid INT NOT NULL REFERENCES users(id),
    movieid INT NOT NULL REFERENCES movies(id),

    stars INT ARRAY[5] DEFAULT '{0, 0, 0, 0, 0}', -- 1:yellow,2:green,3:red,4:blue,5:purple
    last_update TIMESTAMPTZ DEFAULT now(), -- for sorting

    UNIQUE (userid, movieid)
);
CREATE TABLE jumpcodes(
    code TEXT UNIQUE, -- should be about 10^10 unqiue codes, more than enough

    type jump_resource NOT NULL,
    id INT NOT NULL,
    active BOOL DEFAULT TRUE -- use this for 
);

-- add stars to a movie and remove them from the user
CREATE OR REPLACE FUNCTION update_movie_stars(userid INT, movieid INT, color INT, count INT) RETURNS void AS $$
DECLARE
    available INT;
    mauid INT;
BEGIN
    -- IF NOT EXISTS(SELECT 1 FROM user_star WHERE user_star.userid = update_movie_stars.userid AND user_star.movieid = update_movie_stars.movieid) THEN
    --     INSERT INTO user_star (userid, movieid) VALUES (userid, movieid);
    -- END IF;

    IF color > 5 OR color < 1 THEN
        RAISE EXCEPTION 'invalid color';
    END IF;
    
    -- check if the user giving the stars isnt the author
    SELECT author_userid INTO mauid FROM movies WHERE id = update_movie_stars.userid;
    IF mauid = userid THEN
        RAISE EXCEPTION 'author cannot star own movie (big ego ahh)';
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

    -- add stars to movie
    --UPDATE user_star SET stars[color] = stars[color] + count WHERE user_star.userid = update_movie_stars.userid AND user_star.movieid = update_movie_stars.movieid;
    
    INSERT INTO user_star (userid, movieid, stars) VALUES (update_movie_stars.userid, update_movie_stars.movieid, pad_star_arr(color, count))
        ON CONFLICT ON CONSTRAINT user_star_userid_movieid_key DO UPDATE SET stars[color] = user_star.stars[color] + count, last_update = now() WHERE user_star.userid = update_movie_stars.userid AND user_star.movieid = update_movie_stars.movieid;

END;
$$ LANGUAGE plpgsql;

-- all stars a movie has received
CREATE FUNCTION get_movie_stars(movieid INT, OUT tstars INT ARRAY[5]) AS $$
BEGIN
    SELECT ARRAY[coalesce(sum(stars[1]), 0), coalesce(sum(stars[2]), 0), coalesce(sum(stars[3]), 0), coalesce(sum(stars[4]), 0), coalesce(sum(stars[5]), 0)] INTO tstars FROM user_star WHERE user_star.movieid = get_movie_stars.movieid;
END;
$$ LANGUAGE plpgsql;

-- all stars a user has received on their movies
CREATE FUNCTION get_user_stars(userid INT, OUT tstars INT ARRAY[5]) AS $$
BEGIN
    SELECT ARRAY[coalesce(sum(stars[1]), 0), coalesce(sum(stars[2]), 0), coalesce(sum(stars[3]), 0), coalesce(sum(stars[4]), 0), coalesce(sum(stars[5]), 0)] INTO tstars FROM user_star JOIN movies ON movies.author_userid = get_user_stars.userid WHERE user_star.movieid = movies.id AND movies.deleted = false;
END;
$$ LANGUAGE plpgsql;

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
$$ LANGUAGE plpgsql;

CREATE FUNCTION arrsum(arr ANYARRAY, OUT s ANYELEMENT) AS $$
BEGIN
    SELECT sum(a) INTO s FROM unnest(arr) a;
END
$$ LANGUAGE plpgsql;

CREATE FUNCTION pad_star_arr(c INT, n INT, OUT arr INT ARRAY[5]) AS $$
DECLARE
    zer INT ARRAY[4] := '{0, 0, 0, 0}';
BEGIN
    SELECT zer[0:c-1] || ARRAY[n] || zer[0:5-c] INTO arr;
END;
$$ LANGUAGE plpgsql;

-- input jc length for different resources
CREATE FUNCTION random_jumpcode(cl INT, OUT nc TEXT) AS $$
BEGIN
    SELECT array_to_string(ARRAY(SELECT substr('ABXYLRNSWE', floor(random()*cl)::int+1, 1) FROM generate_series(1,cl)), '') INTO nc;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION add_jumpcode_tr_func() RETURNS trigger AS $$
DECLARE
    j_invalid BOOL = true;
    cl INT = 10;
    tn TEXT = TG_TABLE_NAME::regclass::TEXT;
BEGIN
--    IF tn = 'users' THEN
--        cl = 7;
--    ELSE IF tn = 'channels' THEN
--        cl = 5;
--    END IF;
    WHILE j_invalid LOOP
        BEGIN
            INSERT INTO jumpcodes(code, type, id) VALUES (random_jumpcode(cl), trim(trailing 's' from tn)::jump_resource, NEW.id);
            j_invalid = false;
        EXCEPTION
            WHEN unique_violation THEN
                -- try again
        END;
    END LOOP;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER movie_add_jumpcode_tr AFTER INSERT ON movies FOR EACH ROW EXECUTE FUNCTION add_jumpcode_tr_func();
CREATE TRIGGER user_add_jumpcode_tr AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION add_jumpcode_tr_func();
CREATE TRIGGER chan_add_jumpcode_tr AFTER INSERT ON channels FOR EACH ROW EXECUTE FUNCTION add_jumpcode_tr_func();

-- call from SQL_MOVIE_GET_BY_ID to delegate this to the db
-- instead of separate statement
CREATE FUNCTION movie_add_view(movieid INT) RETURNS void AS $$
BEGIN
    UPDATE movies SET views = views + 1 WHERE movieid = movie_add_view.movieid AND deleted = false;
END;
$$ LANGUAGE plpgsql;

COMMIT;