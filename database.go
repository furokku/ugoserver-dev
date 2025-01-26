package main

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

const (
    // these statements never have to change, so they're all here in one place
    // for the sake of accessibility and not having to go look for sql_update_my_balls when
    // something needs to be minorly tweaked

    SQL_MOVIE_ADD string = "INSERT INTO movies (author_userid, author_fsid, author_name, author_filename, lock, channelid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING (id)"
    SQL_MOVIE_DELETE string = "UPDATE movies SET deleted = true WHERE id = $1"
    SQL_MOVIE_CHECK_EXISTS_AFN string = "SELECT EXISTS(SELECT 1 FROM movies WHERE author_filename = $1 AND deleted = false) AS \"EXISTS\""

    SQL_MOVIE_GET_BY_ID string = "WITH movie AS (SELECT * FROM movies WHERE deleted = false AND id = $1), replies AS (SELECT count(1) AS c FROM comments WHERE movieid = $1) SELECT movie.*, yst, gst, rst, bst, pst, replies.c FROM movie, replies, get_movie_stars($1)"
    SQL_MOVIE_GET_NEW string = "WITH filtered AS (SELECT id, yst+gst+rst+bst+pst AS ts FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false ORDER BY uploaded DESC LIMIT 50 OFFSET ($1-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false) SELECT filtered.*, total.t FROM filtered, total"
    SQL_MOVIE_GET_CHANNEL_NEW string = "WITH filtered AS (SELECT id, yst+gst+rst+bst+pst AS ts FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false AND channelid = $1 ORDER BY uploaded DESC LIMIT 50 OFFSET ($2-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false AND channelid = $1) SELECT filtered.*, total.t FROM filtered, total"

    SQL_MOVIE_UPDATE_DL string = "UPDATE movies SET downloads = downloads + 1 WHERE id = $1 AND deleted = false"
    SQL_MOVIE_UPDATE_VIEWS string = "UPDATE movies SET views = views + 1 WHERE id = $1 AND deleted = false"
    
    SQL_CHANNEL_GET_DESC_BY_ID string = "SELECT desc_s, desc_l FROM channels WHERE id = $1"

    SQL_MOVIE_UPDATE_STAR_YELLOW string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS ys) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET ys = target.ys + source.ys WHEN NOT MATCHED THEN INSERT (userid, movieid, ys) VALUES (source.userid, source.movieid, source.ys)"
    SQL_MOVIE_UPDATE_STAR_GREEN string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS gs) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET gs = target.gs + source.gs WHEN NOT MATCHED THEN INSERT (userid, movieid, gs) VALUES (source.userid, source.movieid, source.gs)"
    SQL_MOVIE_UPDATE_STAR_RED string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS rs) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET rs = target.rs + source.rs WHEN NOT MATCHED THEN INSERT (userid, movieid, rs) VALUES (source.userid, source.movieid, source.rs)"
    SQL_MOVIE_UPDATE_STAR_BLUE string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS bs) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET bs = target.bs + source.bs WHEN NOT MATCHED THEN INSERT (userid, movieid, bs) VALUES (source.userid, source.movieid, source.bs)"
    SQL_MOVIE_UPDATE_STAR_PURPLE string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS ps) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET ps = target.ps + source.ps WHEN NOT MATCHED THEN INSERT (userid, movieid, ps) VALUES (source.userid, source.movieid, source.ps)"
    
    SQL_MOVIE_ADD_COMMENT_MEMO string = "INSERT INTO comments (userid, movieid) VALUES ($1, $2) RETURNING (id)" // only need to imply about its existence
    SQL_MOVIE_ADD_COMMENT_TEXT string = "INSERT INTO comments (userid, movieid, is_memo, content) VALUES ($1, $2, false, $3)"
    SQL_MOVIE_GET_COMMENT string = "SELECT comments.*, users.username FROM comments JOIN users ON comments.userid = users.id WHERE movieid = $1 ORDER BY posted DESC LIMIT 10 OFFSET ($2-1)*10"
    
    SQL_WHITELIST_FSID_ADD string = "INSERT INTO auth_whitelist (userfsid) VALUES ($1)"
    SQL_WHITELIST_FSID_DELETE string = "DELETE FROM auth_whitelist WHERE userfsid = $1"
    SQL_WHITELIST_FSID_CHECK string = "SELECT EXISTS(SELECT 1 FROM auth_whitelist WHERE userfsid = $1) AS \"EXISTS\""
    
    SQL_BAN_CHECK string = "SELECT EXISTS(SELECT 1 FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1) AS \"EXISTS\""
    SQL_BAN_QUERY string = "SELECT * FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1"
    SQL_BAN_ISSUE string = "INSERT INTO bans (issuer, expires, message, affected) VALUES ($1, $2, $3, $4)"
    SQL_BAN_PARDON_BY_ID string = "UPDATE bans SET pardon = true WHERE id = $1"
    
    SQL_USER_REGISTER_DSI string = "INSERT INTO users (username, password, fsid, last_login_ip) VALUES ($1, crypt($2, gen_salt('bf')), $3, $4) RETURNING (id)"
    SQL_USER_VERIFY string = "SELECT id FROM users WHERE username = $1 AND password = crypt($2, password)"
    SQL_USER_VERIFY_DSI string = "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND password = crypt($2, password)) AS \"EXISTS\""
    SQL_USER_CHECK_ADMIN string = "SELECT EXISTS(SELECT 1 FROM users WHERE admin = true AND id = $1) AS \"EXISTS\""
    SQL_USER_UPDATE_LAST_LOGIN_IP string = "UPDATE users SET last_login_ip = $2 WHERE id = $1"
    SQL_USER_GET_BY_FSID string = "SELECT id, last_login_ip FROM users WHERE fsid = $1"

    SQL_APITOKEN_SECRET_EXISTS string = "SELECT EXISTS(SELECT 1 FROM apitokens WHERE expires > now() AND secret = crypt($1, secret)) AS \"EXISTS\""
    SQL_APITOKEN_REGISTER string = "INSERT INTO apitokens (userid, secret) VALUES ($1, crypt($2, gen_salt('bf')))"
    SQL_APITOKEN_VERIFY string = "SELECT userid FROM apitokens WHERE expires > now() AND secret = crypt($1, apitokens.secret)"
)

// All of these functions return an error, which is simply
// what DB.Exec or DB.Query returns, if applicable. make sure to handle it correctly

// getMoviesList() returns a list of 50 movies and the total according to the provided SQL statement
func getMoviesList(stmt string, args ...any) ([]Movie, int, error) {

    var memos []Movie
    var t int

    rows, err := db.Query(stmt, args...)
    if err != nil {
        return nil, 0, err
    }

    defer rows.Close()

    for rows.Next() {
        m := Movie{}

        // in a list the star count is a total, so throw that into ys
        if err := rows.Scan(&m.ID, &m.Ys, &t); err != nil { // inefficient: t is read into up to 50 times here
            return nil, 0, err
        }
        memos = append(memos, m)
    }

    return memos, t, nil
}

// getMovieSingle() returns a single movie by ID
func getMovieSingle(movieid int) (Movie, error) {
    var m Movie

    if err := db.QueryRow(SQL_MOVIE_GET_BY_ID, movieid).Scan(&m.ID, &m.AuUserID, &m.AuFSID, &m.AuName, &m.AuFN, &m.Posted, &m.Views, &m.Downloads, &m.Lock, &m.Deleted, &m.ChannelID, &m.Ys, &m.Gs, &m.Gs, &m.Bs, &m.Ps, &m.Replies); err == sql.ErrNoRows {
        return Movie{}, ErrNoMovie
    } else if err != nil {
        return Movie{}, err
    }

    return m, nil
}

// getFrontMovies() returns movies thru getMoviesList based on the mode and page
func getFrontMovies(mode string, p int) ([]Movie, int, error) {
    var q string
    
    switch mode {
    case "new":
        q = SQL_MOVIE_GET_NEW
    default:
        q = SQL_MOVIE_GET_NEW
        warnlog.Printf("tried to get %s movies", mode)
    }

    return getMoviesList(q, p)
}

// getChannelMovies() does the same as getFrontMovies, but from a specific channel
func getChannelMovies(id int, mode string, p int) ([]Movie, int, error) {
    var q string

    switch mode {
    case "new":
        q = SQL_MOVIE_GET_CHANNEL_NEW
    default:
        q = SQL_MOVIE_GET_CHANNEL_NEW
        warnlog.Printf("tried to get %s movies", mode)
    }
    return getMoviesList(q, id, p)
}

// getChannelInfo() will return the short and long description of a channel by ID
func getChannelInfo(id int) (string, string, error) {
    var s, l string

    if err := db.QueryRow(SQL_CHANNEL_GET_DESC_BY_ID, id).Scan(&s, &l); err != nil {
        return "", "", err
    }
    
    return s, l, nil
}

// addMovie() updates the database with information about a movie, returning its ID
func addMovie(userid int, fsid string, name string, fn string, l int, channel int) (int, error) {
    var new_movieid int

    // check if flipnote has already been uploaded
    // using filename (they are always unique)
    if exists, err := checkMovieExistsAfn(fn); err != nil {
        return 0, err
    } else if exists {
        return 0, ErrMovieExists
    }
    
    if err := db.QueryRow(SQL_MOVIE_ADD, userid, fsid, name, fn, l, channel).Scan(&new_movieid); err != nil {
        return 0, err
    }
    
    return new_movieid, nil
}

// deleteMovie() sets a movie as deleted
func deleteMovie(movieid int) error {
    if _, err := db.Exec(SQL_MOVIE_DELETE, movieid); err != nil {
        return err
    }
    return nil
}

// addMovieReplyMemo() updates the database about a new comment and returns its ID
func addMovieReplyMemo(userid int, movieid int) (int, error) {
    var id int
    if err := db.QueryRow(SQL_MOVIE_ADD_COMMENT_MEMO, userid, movieid).Scan(&id); err != nil {
        return 0, err
    }
    return id, nil
}

// getMovieComments() returns the comments on a movie, with pagination
func getMovieComments(movieid int, page int) ([]Comment, error) {
    var comments []Comment

    // comment content
    rows, err := db.Query(SQL_MOVIE_GET_COMMENT, movieid, page)
    if err == sql.ErrNoRows {
        return nil, nil
    } else if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    for rows.Next() {
        c := Comment{}

        if err := rows.Scan(&c.ID, &c.UserID, &c.MovieID, &c.IsMemo, &c.Content, &c.Posted, &c.Username); err != nil {
            return nil, err
        }
        comments = append(comments, c)
    }
    
    return comments, nil
}

// updateMovieStars() updates the database about stars added to a movie
func updateMovieStars(userid int, movieid int, color string, count int) error {
    var q string
    switch color {
    case "yellow":
        q = SQL_MOVIE_UPDATE_STAR_YELLOW
    case "green":
        q = SQL_MOVIE_UPDATE_STAR_GREEN
    case "red":
        q = SQL_MOVIE_UPDATE_STAR_RED
    case "blue":
        q = SQL_MOVIE_UPDATE_STAR_BLUE
    case "purple":
        q = SQL_MOVIE_UPDATE_STAR_PURPLE
    }

    if _, err := db.Exec(q, userid, movieid, count); err != nil {
        return err
    }
    
    return nil
}

// updateViewDlCount() updates the database about views and downloads on a movie
func updateViewDlCount(movieid int, t string) error {
    var q string
    switch t {
    case "dl":
        q = SQL_MOVIE_UPDATE_DL
    case "ppm":
        q = SQL_MOVIE_UPDATE_VIEWS
    }
    if _, err := db.Exec(q, movieid); err != nil {
        return err
    }
    return nil
}

// checkMovieExistsAfn() checks whether a movie has already been uploaded by looking for a movie with the same filename
func checkMovieExistsAfn(afn string) (bool, error) {
    var exists bool

    err := db.QueryRow(SQL_MOVIE_CHECK_EXISTS_AFN, afn).Scan(&exists)
    if err != nil {
        return false, err
    }

    return exists, nil
}

// whitelistAddFsid() adds an FSID to the auth whitelist, so that it immediately passes validation
func whitelistAddFsid(fsid string) error {
    if _, err := db.Exec(SQL_WHITELIST_FSID_ADD, fsid); err != nil {
        return err
    }
    return nil
}

// whitelistDelFsid() removes an FSID from the auth whitelist
func whitelistDelFsid(fsid string) error {
    if _, err := db.Exec(SQL_WHITELIST_FSID_DELETE, fsid); err != nil {
        return err
    }
    return nil
}

// whitelistQueryFsid() checks whether an FSID is in the auth whitelist
func whitelistQueryFsid(fsid string) (bool, error) {
    var v bool
    err := db.QueryRow(SQL_WHITELIST_FSID_CHECK, fsid).Scan(&v)
    if err != nil {
        return false, err
    }
    return v, nil
}

// checkIsBanned() checks whether an IP/FSID is banned and returns true/false
func checkIsBanned(affected string) (bool, error) {
    var exists bool
    err := db.QueryRow(SQL_BAN_CHECK, affected).Scan(&exists)
    if err != nil {
        return false, err
    }
    return exists, nil
}

// queryBan() checks whether an IP/FSID is banned and returns all information about the ban
func queryBan(affected string) (bool, restriction, error) {
    b := restriction{}
    err := db.QueryRow(SQL_BAN_QUERY, affected).Scan(&b.banid, &b.issuer, &b.issued, &b.expires, &b.message, &b.pardon, &b.affected)
    if err == sql.ErrNoRows {
        return false, restriction{}, nil
    } else if err != nil {
        return false, restriction{}, err
    }
    return true, b, nil
}

// issueBan() issues a ban onto a specific IP/FSID, optionally not banning if another ban has not yet expired
func issueBan(iss string, exp time.Time, affected string, msg string, ce bool) error {
    if ce {
        if b, err := checkIsBanned(affected); err != nil {
            return err
        } else if b {
            return ErrAlreadyBanned
        }
    }

    if _, err := db.Exec(SQL_BAN_ISSUE, iss, exp, msg, affected); err != nil {
        return err
    }
    infolog.Printf("%v banned %v until %v for %v", iss, affected, exp, msg)
    return nil
}

// pardonBanId() will pardon a ban by its ID
func pardonBanId(banid int) error {
    if _, err := db.Exec(SQL_BAN_PARDON_BY_ID, banid); err != nil {
        return err
    }
    return nil
}

// registerUserDsi() updates the database and adds a record for a new user
func registerUserDsi(username string, password string, fsid string, ip string) (int, error) {
    var userid int
    if err := db.QueryRow(SQL_USER_REGISTER_DSI, username, password, fsid, ip).Scan(&userid); err != nil {
        return 0, err
    }
    return userid, nil
}

// verifyUserDsi() checks whether a user's password is correct
func verifyUserDsi(userid int, password string) (bool, error) {
    var v bool
    if err := db.QueryRow(SQL_USER_VERIFY_DSI, userid, password).Scan(&v); err != nil {
        return false, err
    }
    return v, nil
}

// getUserDsi() returns a user's ID and last login IP by FSID
func getUserDsi(fsid string) (int, string, error) {
    var userid int
    var last_login_ip string
    if err := db.QueryRow(SQL_USER_GET_BY_FSID, fsid).Scan(&userid, &last_login_ip); err == sql.ErrNoRows {
        return 0, "", ErrNoUser
    } else if err != nil {
        return 0, "", err
    }
    
    return userid, last_login_ip, nil
}

// updateUserLastLogin() updates the user's last login IP
func updateUserLastLogin(userid int, ip string) error {
    if _, err := db.Exec(SQL_USER_UPDATE_LAST_LOGIN_IP, userid, ip); err != nil {
        return err
    }
    return nil
}

// registerApiToken() will create a unique token for API access
// Not implemented yet
func registerApiToken(userid int) error {
    var exists bool
    var secret string
    
    for {
        secret = randAsciiString(72)
        if err := db.QueryRow(SQL_APITOKEN_SECRET_EXISTS, secret).Scan(&exists); err != nil {
            return err
        }
        if !exists {
            break
        }
    }
    
    if _, err := db.Exec(SQL_APITOKEN_REGISTER, userid, secret); err != nil {
        return err
    }
    return nil
}