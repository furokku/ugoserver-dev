-- made for PostgreSQL. probably won't work on mysql/sqlite!
BEGIN;
CREATE EXTENSION pgcrypto;
CREATE TABLE flipnotes(
    id SERIAL PRIMARY KEY,
    author_id VARCHAR(16) NOT NULL,
    author_name TEXT NOT NULL,
    parent_author_id VARCHAR(16) NOT NULL,
    parent_author_name TEXT NOT NULL,
    author_filename VARCHAR(24) NOT NULL,
    uploaded_at TIMESTAMP DEFAULT now(),
    views INTEGER DEFAULT 0,
    downloads INTEGER DEFAULT 0,
    yellow_stars INTEGER DEFAULT 0,
    green_stars INTEGER DEFAULT 0,
    red_stars INTEGER DEFAULT 0,
    blue_stars INTEGER DEFAULT 0,
    purple_stars INTEGER DEFAULT 0,
    lock BOOL DEFAULT FALSE,
    deleted BOOL DEFAULT FALSE,
    channelid INTEGER DEFAULT 0
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
    reason TEXT DEFAULT 'why not lmao',
    message TEXT DEFAULT 'begone',
    pardon BOOL DEFAULT FALSE,
    affected TEXT NOT NULL
);
CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password VARCHAR(60) NOT NULL,
    admin BOOL DEFAULT FALSE,
    fsid VARCHAR(16) DEFAULT '0000000000000000'
);
CREATE TABLE apitokens(
    id SERIAL PRIMARY KEY,
    secret VARCHAR(60) UNIQUE NOT NULL,
    expires TIMESTAMPTZ DEFAULT now() + interval '30 days',
    userid INTEGER NOT NULL
);
COMMIT;