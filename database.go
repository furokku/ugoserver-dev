package main

import (
	"database/sql"
	"fmt"
	"time"
)

const (
    SQL_MOVIE_ADD string = "INSERT INTO movies (author_userid, author_fsid, author_name, author_filename, lock) VALUES ($1, $2, $3, $4, $5) RETURNING (id)"
    SQL_MOVIE_DELETE string = "UPDATE movies SET deleted = true WHERE id = $1"
    SQL_MOVIE_COUNT string = "SELECT count(1) FROM movies WHERE deleted = false"
    SQL_MOVIE_GET_BY_ID string = "SELECT * FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false AND id = $1"
    SQL_MOVIE_GET_RECENT string = "SELECT id, yst+gst+rst+bst+pst AS ts FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false ORDER BY uploaded DESC LIMIT 50 OFFSET ($1-1)*50"
    SQL_MOVIE_UPDATE_DL string = "UPDATE movies SET downloads = downloads + 1 WHERE id = $1"
    SQL_MOVIE_UPDATE_VIEWS string = "UPDATE movies SET views = views + 1 WHERE id = $1"
    SQL_MOVIE_CHECK_EXISTS_AFN string = "SELECT EXISTS(SELECT 1 FROM movies WHERE author_filename = $1) AS \"EXISTS\""

    SQL_MOVIE_UPDATE_USER_STAR_YELLOW string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS ys) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET ys = target.ys + source.ys WHEN NOT MATCHED THEN INSERT (userid, movieid, ys) VALUES (source.userid, source.movieid, source.ys)"
    SQL_MOVIE_UPDATE_USER_STAR_GREEN string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS gs) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET gs = target.gs + source.gs WHEN NOT MATCHED THEN INSERT (userid, movieid, gs) VALUES (source.userid, source.movieid, source.gs)"
    SQL_MOVIE_UPDATE_USER_STAR_RED string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS rs) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET rs = target.rs + source.rs WHEN NOT MATCHED THEN INSERT (userid, movieid, rs) VALUES (source.userid, source.movieid, source.rs)"
    SQL_MOVIE_UPDATE_USER_STAR_BLUE string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS bs) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET bs = target.bs + source.bs WHEN NOT MATCHED THEN INSERT (userid, movieid, bs) VALUES (source.userid, source.movieid, source.bs)"
    SQL_MOVIE_UPDATE_USER_STAR_PURPLE string = "MERGE INTO user_star AS target USING (SELECT CAST($1 AS INTEGER) AS userid, CAST($2 AS INTEGER) AS movieid, CAST($3 AS INTEGER) AS ps) AS source ON target.userid = source.userid AND target.movieid = source.movieid WHEN MATCHED THEN UPDATE SET ps = target.ps + source.ps WHEN NOT MATCHED THEN INSERT (userid, movieid, ps) VALUES (source.userid, source.movieid, source.ps)"
    
    SQL_WHITELIST_FSID_ADD string = "INSERT INTO auth_whitelist (fsid) VALUES ($1)"
    SQL_WHITELIST_FSID_DELETE string = "DELETE FROM auth_whitelist WHERE fsid = $1"
    SQL_WHITELIST_FSID_CHECK string = "SELECT EXISTS(SELECT 1 FROM auth_whitelist WHERE fsid = $1) AS \"EXISTS\""
    
    SQL_BAN_CHECK string = "SELECT EXISTS(SELECT 1 FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1) AS \"EXISTS\""
    SQL_BAN_QUERY string = "SELECT * FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1"
    SQL_BAN_ISSUE string = "INSERT INTO bans (issuer, expires, reason, message, affected) VALUES ($1, $2, $3, $4, $5)"
    SQL_BAN_PARDON_BY_ID string = "UPDATE bans SET pardon = true WHERE id = $1"
    
    SQL_USER_REGISTER string = "INSERT INTO users (username, password, fsid) VALUES ($1, crypt($2, gen_salt('bf')), fsid) RETURNING (id)"
    SQL_USER_VERIFY string = "SELECT id FROM users WHERE username = $1 AND password = crypt($2, password)"
    SQL_USER_VERIFY_DSI string = "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND password = crypt($2, password)) AS \"EXISTS\""
    SQL_USER_CHECK_ADMIN string = "SELECT EXISTS(SELECT 1 FROM users WHERE admin = true AND id = $1) AS \"EXISTS\""
    SQL_USER_UPDATE_LAST_LOGIN_IP string = "UPDATE users WHERE id = $1 SET last_login_ip = $2"
    SQL_USER_GET_BY_FSID string = "SELECT id, last_login_ip FROM users WHERE fsid = $1"

    SQL_APITOKEN_SECRET_EXISTS string = "SELECT EXISTS(SELECT 1 FROM apitokens WHERE expires > now() AND secret = crypt($1, secret)) AS \"EXISTS\""
    SQL_APITOKEN_REGISTER string = "INSERT INTO apitokens (userid, secret) VALUES ($1, crypt($2, gen_salt('bf')))"
    SQL_APITOKEN_VERIFY string = "SELECT userid FROM apitokens WHERE expires > now() AND secret = crypt($1, apitokens.secret)"
)

func connect() (*sql.DB, error) {
    var cs string

    switch cnf.DB.Type {
    // only postgres is supported for now
    case "postgres":
        cs = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cnf.DB.Host, cnf.DB.Port, cnf.DB.User, cnf.DB.Pass, cnf.DB.Name)
    default:
        return nil, ErrInvalidDbType
    }
    
    db, err := sql.Open(cnf.DB.Type, cs)
    if err != nil {
        return nil, err
    }
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return db, nil
}

// All of these functions return an error, which is simply
// what DB.Exec or DB.Query returns, if applicable. Handle it!

// return flipnotes for sql statement
func getMoviesList(stmt string, args ...any) ([]flipnote, error) {

    var memos []flipnote

    rows, err := db.Query(stmt, args...)
    if err != nil {
        return []flipnote{}, err
    }

    defer rows.Close()

    for rows.Next() {
        m := flipnote{}

        // in a list the star count is a total, so throw that into ys
        if err := rows.Scan(&m.id, &m.ys); err != nil {
            return []flipnote{}, err
        }
        memos = append(memos, m)
    }

    return memos, nil
}

func getMovieSingle(movieid int) (flipnote, error) {
    var m flipnote

    if err := db.QueryRow(SQL_MOVIE_GET_BY_ID, movieid).Scan(&m.id, &m.author_userid, &m.author_fsid, &m.author_name, &m.author_filename, &m.uploaded, &m.views, &m.downloads, &m.lock, &m.deleted, &m.channelid, &m.ys, &m.gs, &m.rs, &m.bs, &m.ps); err != nil {
        return flipnote{}, err
    }

    return m, nil
}

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

func addMovie(userid int, fsid string, name string, fn string, l int) (int, error) {
    var new_movieid int

    // check if flipnote has already been uploaded
    // using filename (they are always unique)
    if exists, err := checkMovieExistsAfn(fn); err != nil {
        return 0, err
    } else if exists {
        return 0, ErrMovieExists
    }
    
    if err := db.QueryRow(SQL_MOVIE_ADD, userid, fsid, name, fn, l).Scan(&new_movieid); err != nil {
        return 0, err
    }
    
    return new_movieid, nil
}

func deleteMovie(movieid int) error {
    if _, err := db.Exec(SQL_MOVIE_DELETE, movieid); err != nil {
        return err
    }
    return nil
}

func updateMovieStars(userid int, movieid int, color string, count int) error {
    var q string
    switch color {
    case "yellow":
        q = SQL_MOVIE_UPDATE_USER_STAR_YELLOW
    case "green":
        q = SQL_MOVIE_UPDATE_USER_STAR_GREEN
    case "red":
        q = SQL_MOVIE_UPDATE_USER_STAR_RED
    case "blue":
        q = SQL_MOVIE_UPDATE_USER_STAR_BLUE
    case "purple":
        q = SQL_MOVIE_UPDATE_USER_STAR_PURPLE
    }

    if _, err := db.Exec(q, userid, movieid, count); err != nil {
        return err
    }
    
    return nil
}

func getFrontMovies(ptype string, p int) ([]flipnote, int, error) {
    var total int
    var q string
    
    switch ptype {
    case "recent":
        q = SQL_MOVIE_GET_RECENT
    default:
        q = SQL_MOVIE_GET_RECENT
        errorlog.Printf("tried to get %s movies", ptype)
    }

    // Get total amount of flipnotes for pagination and top screen text
    if err := db.QueryRow(SQL_MOVIE_COUNT).Scan(&total); err != nil {
        return []flipnote{}, 0, err
    }

    movies, err := getMoviesList(q, p)
    if err != nil {
        return []flipnote{}, 0, err
    }
    
    return movies, total, nil
}

func checkMovieExistsAfn(afn string) (bool, error) {
    var exists bool

    err := db.QueryRow(SQL_MOVIE_CHECK_EXISTS_AFN, afn).Scan(&exists)
    if err != nil {
        return false, err
    }

    return exists, nil
}

func whitelistAddFsid(fsid string) error {
    if _, err := db.Exec(SQL_WHITELIST_FSID_ADD, fsid); err != nil {
        return err
    }
    return nil
}

func whitelistDelFsid(fsid string) error {
    if _, err := db.Exec(SQL_WHITELIST_FSID_DELETE, fsid); err != nil {
        return err
    }
    return nil
}

func whitelistQueryFsid(fsid string) (bool, error) {
    var v bool
    err := db.QueryRow(SQL_WHITELIST_FSID_CHECK, fsid).Scan(&v)
    if err != nil {
        return false, err
    }
    return v, nil
}

// returns only true/false
func checkIsBanned(affected string) (bool, error) {
    var exists bool
    err := db.QueryRow(SQL_BAN_CHECK, affected).Scan(&exists)
    if err != nil {
        return false, err
    }
    return exists, nil
}

// returns whole ban
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

func issueBan(iss string, exp time.Time, affected string, r string, msg string, ce bool) error {
    if ce {
        if b, err := checkIsBanned(affected); err != nil {
            return err
        } else if b {
            return ErrAlreadyBanned
        }
    }

    if _, err := db.Exec(SQL_BAN_ISSUE, iss, exp, r, msg, affected); err != nil {
        return err
    }
    infolog.Printf("%v banned %v until %v for %v (%v)", iss, affected, exp, r, msg)
    return nil
}

func pardonBanId(banid int) error {
    if _, err := db.Exec(SQL_BAN_PARDON_BY_ID, banid); err != nil {
        return err
    }
    return nil
}

func registerUser(username string, password string, fsid string) (int, error) {
    var userid int
    if err := db.QueryRow(SQL_USER_REGISTER, username, password, fsid).Scan(&userid); err != nil {
        return 0, err
    }
    return userid, nil
}

func verifyUserDsi(userid int, password string) (bool, error) {
    var v bool
    if err := db.QueryRow(SQL_USER_VERIFY_DSI, userid, password).Scan(&v); err != nil {
        return false, err
    }
    return v, nil
}

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

func updateUserLastLogin(userid int, ip string) error {
    if _, err := db.Exec(SQL_USER_UPDATE_LAST_LOGIN_IP, userid, ip); err != nil {
        return err
    }
    return nil
}

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