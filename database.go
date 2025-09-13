package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
    // these statements never have to change, so they're all here in one place
    // for the sake of accessibility and not having to go look for sql_update_my_balls when
    // something needs to be minorly tweaked

    SQL_MOVIE_ADD string = "INSERT INTO movies (channelid, author_userid, author_fsid, author_name, author_filename, og_author_fsid, og_author_name, og_author_filename_fragment, lock, last_modified) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING (id)"
    SQL_MOVIE_DELETE string = "UPDATE movies SET deleted = true WHERE id = $1 AND deleted = false"
    SQL_MOVIE_CHECK_EXISTS_AFN string = "SELECT EXISTS(SELECT TRUE movies WHERE author_filename = $1 AND deleted = false)"

    // hard to read but whatev
    SQL_MOVIE_GET_BY_ID string = "WITH movie AS (SELECT * FROM movies WHERE deleted = false AND id = $1), replies AS (SELECT count(1) AS c FROM comments WHERE movieid = $1), jump AS (SELECT code FROM jumpcodes WHERE type = 'movie' AND id = $1), uv AS (SELECT movie_add_view($1)) SELECT movie.*, tstars, replies.c, code FROM movie, replies, get_movie_stars($1), jump JOIN uv ON 1 = 1"

    SQL_MOVIE_GET_NEW string = "WITH filtered AS (SELECT id, arrsum(tstars) AS ts FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false ORDER BY posted DESC LIMIT 50 OFFSET ($1-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false) SELECT filtered.id, filtered.ts, total.t FROM filtered, total"
    SQL_MOVIE_GET_TOP string = "WITH filtered AS (SELECT id, arrsum(tstars) AS ts FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false ORDER BY ts DESC LIMIT 50 OFFSET ($1-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false) SELECT filtered.id, filtered.ts, total.t FROM filtered, total" // seems fine, might limit timeframe at some point
    SQL_MOVIE_GET_HOT string = "WITH filtered AS (SELECT id, arrsum(tstars) AS ts, date_bin('7 days', movies.posted, TIMESTAMP '2025-08-23') AS bin FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false ORDER BY bin DESC, ts DESC, posted DESC LIMIT 50 OFFSET ($1-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false) SELECT filtered.id, filtered.ts, total.t FROM filtered, total"
    SQL_MOVIE_GET_CHANNEL_NEW string = "WITH filtered AS (SELECT id, arrsum(tstars) AS ts FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false AND channelid = $2 ORDER BY posted DESC LIMIT 50 OFFSET ($1-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false AND channelid = $2) SELECT filtered.id, filtered.ts, total.t FROM filtered, total"
    SQL_MOVIE_GET_CHANNEL_TOP string = "WITH filtered AS (SELECT id, arrsum(tstars) AS ts FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false AND channelid = $2 ORDER BY ts DESC LIMIT 50 OFFSET ($1-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false AND channelid = $2) SELECT filtered.id, filtered.ts, total.t FROM filtered, total" // seems fine, might limit timeframe at some point
    SQL_MOVIE_GET_CHANNEL_HOT string = "WITH filtered AS (SELECT id, arrsum(tstars) AS ts, date_bin('7 days', movies.posted, TIMESTAMP '2025-08-23') AS bin FROM movies JOIN get_movie_stars(id) ON TRUE WHERE deleted = false AND channelid = $2 ORDER BY bin DESC, ts DESC, posted DESC LIMIT 50 OFFSET ($1-1)*50), total AS (SELECT count(1) AS t FROM movies WHERE deleted = false AND channelid = $2) SELECT filtered.id, filtered.ts, total.t FROM filtered, total"

    SQL_MOVIE_UPDATE_DL string = "UPDATE movies SET downloads = downloads + 1 WHERE id = $1 AND deleted = false"
    // moved into MOVIE_GET_BY_ID
    //SQL_MOVIE_UPDATE_VIEWS string = "UPDATE movies SET views = views + 1 WHERE id = $1 AND deleted = false"
    
    SQL_CHANNEL_CREATE string = "INSERT INTO channels (chname, dsc) VALUES ($1, $2) RETURNING id"
    SQL_CHANNEL_DELETE string = "UPDATE channels SET deleted = true WHERE id = $1 AND deleted = false"
    SQL_CHANNEL_RENAME string = "UPDATE channels SET chname = $2 WHERE id = $1 AND deleted = false"
    SQL_CHANNEL_UPDATE string = "UPDATE channels SET dsc = $2 WHERE id = $1 AND deleted = false"
    SQL_CHANNEL_GET_MAIN string = "SELECT id, chname FROM channels WHERE deleted = false ORDER BY id ASC LIMIT 8"
    SQL_CHANNEL_GET_MORE string = "SELECT id, chname FROM channels WHERE deleted = false ORDER BY id ASC OFFSET 8+($1-1)*15 LIMIT 15" // tweak limit if needed; pagination
    SQL_CHANNEL_GET_DESC_BY_ID string = "SELECT chname, dsc FROM channels WHERE id = $1 AND deleted = false"

    SQL_MOVIE_UPDATE_STARS string = "SELECT update_movie_stars($1, $2, $3, $4)"
    SQL_USER_GET_EXPENDABLE_STARS string = "SELECT expendable_stars FROM users WHERE id = $1"
    SQL_USER_GET_STARS string = "SELECT tstars FROM get_user_stars($1)"

    SQL_COMMENT_ADD_MEMO string = "INSERT INTO comments (userid, movieid) VALUES ($1, $2) RETURNING (id)" // only need to imply about its existence
    SQL_COMMENT_ADD_TEXT string = "INSERT INTO comments (userid, movieid, is_memo, content) VALUES ($1, $2, false, $3)"
    SQL_COMMENT_GET_ON_MOVIE string = "SELECT comments.*, users.username FROM comments JOIN users ON comments.userid = users.id WHERE movieid = $1 AND comments.deleted = false ORDER BY posted ASC LIMIT 10 OFFSET ($2-1)*10"
    SQL_COMMENT_DELETE string = "UPDATE comments SET deleted = true WHERE id = $1 and deleted = false"
    
    SQL_WHITELIST_FSID_ADD string = "INSERT INTO auth_whitelist (fsid) VALUES ($1)"
    SQL_WHITELIST_FSID_DELETE string = "DELETE FROM auth_whitelist WHERE fsid = $1"
    SQL_WHITELIST_FSID_CHECK string = "SELECT EXISTS(SELECT TRUE FROM auth_whitelist WHERE fsid = $1)"
    
    SQL_USER_BAN_CHECK string = "SELECT EXISTS(SELECT TRUE FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1)"
    SQL_USER_BAN_QUERY string = "SELECT * FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1"
    SQL_USER_BAN string = "INSERT INTO bans (issuer, expires, message, affected) VALUES ($1, $2, $3, $4)"
    SQL_USER_PARDON string = "UPDATE bans SET pardon = true WHERE id IN(SELECT max(id) FROM bans WHERE affected = $1 AND pardon = false)"
    SQL_USER_PARDON_BY_ID string = "UPDATE bans SET pardon = true WHERE id = $1"
    
    SQL_USER_REGISTER_DSI string = "INSERT INTO users (username, password, fsid, last_login_ip) VALUES ($1, crypt($2, gen_salt('bf')), $3, $4) RETURNING (id)"
    //SQL_USER_VERIFY string = "SELECT id FROM users WHERE username = $1 AND password = crypt($2, password) AND deleted = false"
    SQL_USER_VERIFY_BY_ID string = "SELECT EXISTS(SELECT TRUE FROM users WHERE id = $1 AND password = crypt($2, password) AND deleted = false)"
    SQL_USER_VERIFY_BY_FSID string = "SELECT EXISTS(SELECT TRUE FROM users WHERE fsid = $1 AND password = crypt($2, password) AND deleted = false)"
    SQL_USER_UPDATE_LAST_LOGIN_BY_ID string = "UPDATE users SET last_login_ip = $2, last_login_time = now() WHERE id = $1"
    SQL_USER_UPDATE_LAST_LOGIN_BY_FSID string = "UPDATE users SET last_login_ip = $2, last_login_time = now() WHERE fsid = $1"
    SQL_USER_CHECK_ADMIN string = "SELECT EXISTS(SELECT TRUE FROM users WHERE admin = true AND id = $1 AND deleted = false)"
    SQL_USER_GET_BY_ID string = "SELECT id, username, admin, fsid, last_login_time, last_login_ip, expendable_stars FROM users WHERE id = $1 AND deleted = false"
    SQL_USER_GET_BY_FSID string = "SELECT id, username, admin, fsid, last_login_time, last_login_ip, expendable_stars FROM users WHERE fsid = $1 AND deleted = false"
    SQL_USER_GET_BY_TOKEN string = "SELECT users.id, username, admin, fsid, last_login_time, last_login_ip, expendable_stars FROM users JOIN apitokens ON users.id = apitokens.userid WHERE apitokens.secret = crypt($1, secret) AND users.deleted = false"
    SQL_USER_RATELIMIT string = "SELECT * FROM get_user_ratelimit($1)"
    
    SQL_JUMPCODE_GET string = "SELECT type, id FROM jumpcodes WHERE code = $1 AND active = true"
    SQL_JUMPCODE_SET_INACTIVE string = "UPDATE jumpcodes SET active = false WHERE type = $1 AND id = $2"

    SQL_APITOKEN_SECRET_EXISTS string = "SELECT EXISTS(SELECT TRUE FROM apitokens WHERE secret = crypt($1, secret))"
    SQL_APITOKEN_REGISTER string = "INSERT INTO apitokens (userid, secret) VALUES ($1, crypt($2, gen_salt('bf')))"
    SQL_APITOKEN_GET_USERID string = "SELECT userid FROM apitokens WHERE secret = crypt($1, secret)"
    SQL_APITOKEN_DESTROY string = "DELETE FROM apitokens WHERE secret = crypt($1, secret)"
    SQL_APITOKEN_GET_ALT string = "SELECT id, created FROM apitokens WHERE userid = $1 AND secret = crypt($1, secret)"
)

// functions use dbhandle to be connection-agnostic
// they will work on a conn, pool, tx, etc

//
// MOVIE
//

// getMoviesList() returns a list of 50 movies and the total according to the provided SQL statement
func getMoviesList(db dbhandle, stmt string, args ...any) ([]Movie, int, error) {

    var memos []Movie
    var t int

    rows, err := db.Query(context.Background(), stmt, args...)
    if err != nil {
        return nil, 0, err
    }

    defer rows.Close()

    for rows.Next() {
        m := Movie{}
        var s int

        // in a list the star count is a total, so throw that into ys
        if err := rows.Scan(&m.ID, &s, &t); err != nil {
            // t is read into like 50 times here
            // but still better than a separate statement for each query
            return nil, 0, err
        }
        m.Stars = []int{s}
        memos = append(memos, m)
    }

    return memos, t, nil
}

func getMovieById(db dbhandle, movieid int) (Movie, error) {
    var m Movie
    s := make([]int, 5)

    if err := db.QueryRow(context.Background(), SQL_MOVIE_GET_BY_ID, movieid).Scan(&m.ID, &m.ChannelID, &m.AuUserID, &m.AuFSID, &m.AuName, &m.AuFN, &m.OGAuFSID, &m.OGAuName, &m.OGAuFNFrag, &m.Views, &m.Downloads, &m.Lock, &m.Deleted, &m.Posted, &m.LastMod, &s, &m.Replies, &m.JumpCode); err != nil {
        switch err {
        case pgx.ErrNoRows:
            return Movie{}, ErrNoMovie
        default:
            return Movie{}, err
        }
    }

    m.Stars = s
    return m, nil
}

// returns movies thru getMoviesList based on the mode and page
func getFrontMovies(db dbhandle, sm string, p int) ([]Movie, int, error) {
    var q string
    
    switch sm { // sort mode
    case "new":
        q = SQL_MOVIE_GET_NEW
    case "hot":
        q = SQL_MOVIE_GET_HOT
    case "top":
        q = SQL_MOVIE_GET_TOP
    default:
        q = SQL_MOVIE_GET_NEW
        warnlog.Printf("invalid sort mode %s", sm)
    }

    return getMoviesList(db, q, p)
}

//  does the same as getFrontMovies, but from a specific channel
func getChannelMovies(db dbhandle, id int, sm string, p int) ([]Movie, int, error) {
    var q string

    switch sm {
    case "new":
        q = SQL_MOVIE_GET_CHANNEL_NEW
    case "hot":
        q = SQL_MOVIE_GET_CHANNEL_HOT
    case "top":
        q = SQL_MOVIE_GET_CHANNEL_TOP
    default:
        q = SQL_MOVIE_GET_CHANNEL_NEW
        warnlog.Printf("invalid sort mode %s", sm)
    }
    return getMoviesList(db, q, p, id)
}

// add record for a movie
func addMovie(db dbhandle, m Movie) (int, error) {
    var new_movieid int

    // check if flipnote has already been uploaded
    // using filename (they are always unique)
    if exists, err := checkMovieExistsAfn(db, m.AuFN); err != nil {
        return 0, err
    } else if exists {
        return 0, ErrMovieExists
    }
    
    if err := db.QueryRow(context.Background(), SQL_MOVIE_ADD, m.ChannelID, m.AuUserID, m.AuFSID, m.AuName, m.AuFN, m.OGAuFSID, m.OGAuName, m.OGAuFNFrag, m.Lock, m.LastMod).Scan(&new_movieid); err != nil {
        return 0, err
    }
    
    return new_movieid, nil
}

// mark movie as deleted
func deleteMovie(db dbhandle, movieid int) error {
    if _, err := db.Exec(context.Background(), SQL_MOVIE_DELETE, movieid); err != nil {
        return err
    }
    return nil
}

// update # of downloads on movie
func updateDlCount(db dbhandle, movieid int) error {
    if _, err := db.Exec(context.Background(), SQL_MOVIE_UPDATE_DL, movieid); err != nil {
        return err
    }
    return nil
}

// checks whether a movie has already been uploaded by looking for a movie with the same filename
func checkMovieExistsAfn(db dbhandle, afn string) (bool, error) {
    var exists bool

    err := db.QueryRow(context.Background(), SQL_MOVIE_CHECK_EXISTS_AFN, afn).Scan(&exists)
    if err != nil {
        return false, err
    }

    return exists, nil
}


//
// CHANNEL
//

// return the name and description of a channel by its ID
func getChannelInfo(db dbhandle, id int) (string, string, error) {
    var s, l string

    if err := db.QueryRow(context.Background(), SQL_CHANNEL_GET_DESC_BY_ID, id).Scan(&s, &l); err != nil {
        return "", "", err
    }
    
    return s, l, nil
}

// returns a list of channels, page 0 - first 8, >0 - paginated x15
func getChannelList(db dbhandle, page int) ([]Channel, error) {
    var rows pgx.Rows
    var err error

    chcap := 15

    if page == 0 {
        chcap = 8
        rows, err = db.Query(context.Background(), SQL_CHANNEL_GET_MAIN)
    } else {
        rows, err = db.Query(context.Background(), SQL_CHANNEL_GET_MORE, page)
    }

    ch := make([]Channel, 0, chcap)

    if err != nil {
        return nil, err
    }
    
    for rows.Next() {
        var id int
        var name string
        
        rows.Scan(&id, &name)
        ch = append(ch, Channel{ID: id, Name: name})
    }

    return ch, nil
}


//
// REPLIES
//

// for drawn comments we only need to imply about its existence
func addMovieCommentMemo(db dbhandle, userid int, movieid int) (int, error) {
    var id int
    if err := db.QueryRow(context.Background(), SQL_COMMENT_ADD_MEMO, userid, movieid).Scan(&id); err != nil {
        return 0, err
    }
    return id, nil
}

// returns the comments on a movie, with pagination
func getMovieComments(db dbhandle, movieid int, page int) ([]Comment, error) {
    var comments []Comment

    // comment content
    rows, err := db.Query(context.Background(), SQL_COMMENT_GET_ON_MOVIE, movieid, page)
    if err != nil {
        switch err {
        case pgx.ErrNoRows:
            return nil, nil
        default:
            return nil, err
        }
    }
    defer rows.Close()
    
    for rows.Next() {
        c := Comment{}

        if err := rows.Scan(&c.ID, &c.UserID, &c.MovieID, &c.IsMemo, &c.Content, &c.Posted, &c.Deleted, &c.Username); err != nil {
            return nil, err
        }
        comments = append(comments, c)
    }
    
    return comments, nil
}


//
// STARS
//

// add stars to a movie
func updateMovieStars(db dbhandle, userid int, movieid int, color string, count int) error {
    var c int
    switch color {
    case "yellow":
        c = 1
    case "green":
        c = 2
    case "red":
        c = 3
    case "blue":
        c = 4
    case "purple":
        c = 5
    }

    if _, err := db.Exec(context.Background(), SQL_MOVIE_UPDATE_STARS, userid, movieid, c, count); err != nil {
        return err
    }
    
    return nil
}

// get stars a user has received across all of their movies
func getUserStars(db dbhandle, userid int) ([]int, error) {
    s := make([]int, 5)
    
    if err := db.QueryRow(context.Background(), SQL_USER_GET_STARS, userid).Scan(&s); err != nil {
        return nil, err
    }
    
    return s, nil
}


//
// WHITELIST
//

// adds an FSID to the auth whitelist, so that it immediately passes validation
func whitelistAddFsid(db dbhandle, fsid string) error {
    if _, err := db.Exec(context.Background(), SQL_WHITELIST_FSID_ADD, fsid); err != nil {
        return err
    }
    return nil
}

// removes an FSID from the auth whitelist
func whitelistDelFsid(db dbhandle, fsid string) error {
    if _, err := db.Exec(context.Background(), SQL_WHITELIST_FSID_DELETE, fsid); err != nil {
        return err
    }
    return nil
}

// checks whether an FSID is in the auth whitelist
func whitelistQueryFsid(db dbhandle, fsid string) (bool, error) {
    var v bool
    err := db.QueryRow(context.Background(), SQL_WHITELIST_FSID_CHECK, fsid).Scan(&v)
    if err != nil {
        return false, err
    }
    return v, nil
}


//
// BAN
//

// returns only bool whether banned
func checkIsBanned(db dbhandle, affected string) (bool, error) {
    var exists bool
    err := db.QueryRow(context.Background(), SQL_USER_BAN_CHECK, affected).Scan(&exists)
    if err != nil {
        return false, err
    }
    return exists, nil
}

// returns bool whether banned and ban info
func queryBan(db dbhandle, affected string) (*Ban, error) {
    b := Ban{}
    err := db.QueryRow(context.Background(), SQL_USER_BAN_QUERY, affected).Scan(&b.ID, &b.Issuer, &b.Issued, &b.Expires, &b.Message, &b.Pardon, &b.Affected)
    if err != nil {
        switch err {
        case pgx.ErrNoRows:
            return nil, nil
        default:
            return nil, err
        }
    }
    return &b, nil
}

// ban a specific IP/FSID, optionally not banning if another ban has not yet expired
func issueBan(db dbhandle, iss string, exp time.Time, affected string, msg string, ce bool) error {
    if ce {
        if b, err := checkIsBanned(db, affected); err != nil {
            return err
        } else if b {
            return ErrAlreadyBanned
        }
    }

    if _, err := db.Exec(context.Background(), SQL_USER_BAN, iss, exp, msg, affected); err != nil {
        return err
    }
    infolog.Printf("%v banned %v until %v for %v", iss, affected, exp, msg)
    return nil
}

// pardon a ban by its ID when given an integer argument
// and the latest active ban on an FSID or IP when given a string argument
func pardonBan(db dbhandle, in any) error {
    switch in.(type) {
    case int:
        if _, err := db.Exec(context.Background(), SQL_USER_PARDON_BY_ID, in); err != nil {
            return err
        }
    case string:
        if _, err := db.Exec(context.Background(), SQL_USER_PARDON, in); err != nil {
            return err
        }
    }
    return nil
}


//
// USER
//

// (use only on dsi part) adds a record for the new user and returns its id
func registerUserDsi(db dbhandle, username string, password string, fsid string, ip string) (int, error) {
    var userid int
    if err := db.QueryRow(context.Background(), SQL_USER_REGISTER_DSI, username, password, fsid, ip).Scan(&userid); err != nil {
        return 0, err
    }
    return userid, nil
}

// returns true/false if the user's password matches based on id
func verifyUserById(db dbhandle, userid int, password string, ip... string) (bool, error) {
    v := false
    if err := db.QueryRow(context.Background(), SQL_USER_VERIFY_BY_ID, userid, password).Scan(&v); err != nil {
        return false, err
    }
    if len(ip)>0 {
        if _, err := db.Exec(context.Background(), SQL_USER_UPDATE_LAST_LOGIN_BY_ID, userid, ip[0]); err != nil {
            return false, err
        }
    }
    return v, nil
}

// TODO: consolidate?
// returns true/false if the user's password matches based on fsid (alternative login method)
func verifyUserByFsid(db dbhandle, fsid string, password string, ip... string) (bool, error) {
    v := false
    if err := db.QueryRow(context.Background(), SQL_USER_VERIFY_BY_FSID, fsid, password).Scan(&v); err != nil {
        return false, err
    }
    if len(ip)>0 {
        if _, err := db.Exec(context.Background(), SQL_USER_UPDATE_LAST_LOGIN_BY_FSID, fsid, ip[0]); err != nil {
            return false, err            
        }
    }
    return v, nil
}


// returns a user by fsid (returns uninitialized array if not found)
func getUserByFsid(db dbhandle, fsid string) (User, error) {
    u := User{}
    es := make([]int, 5)
    if err := db.QueryRow(context.Background(), SQL_USER_GET_BY_FSID, fsid).Scan(&u.ID, &u.Username, &u.Admin, &u.FSID, &u.LastLoginTime, &u.LastLoginIP, &es); err != nil {
        switch err {
        case pgx.ErrNoRows:
            return User{}, ErrNoUser
        default:
            return User{}, err
        }
    }
    
    u.ExpendableStars = es
    
    return u, nil
}

// same but for id
func getUserById(db dbhandle, id int) (User, error) {
    u := User{}
    es := make([]int, 5)
    if err := db.QueryRow(context.Background(), SQL_USER_GET_BY_ID, id).Scan(&u.ID, &u.Username, &u.Admin, &u.FSID, &u.LastLoginTime, &u.LastLoginIP, &es); err != nil {
        switch err {
        case pgx.ErrNoRows:
            return User{}, ErrNoUser
        default:
            return User{}, err
        }
    }
    
    u.ExpendableStars = es
    return u, nil
}

func getUserApiToken(db dbhandle, secret string) (User, error) {
    u := User{}
    es := make([]int, 5)

    if err := db.QueryRow(context.Background(), SQL_USER_GET_BY_TOKEN, secret).Scan(&u.ID, &u.Username, &u.Admin, &u.FSID, &u.LastLoginTime, &u.LastLoginIP, &es); err != nil {
        switch err {
        case pgx.ErrNoRows:
            return User{}, ErrNoUser
        default:
            return User{}, err
        }
    }
    
    u.ExpendableStars = es
    return u, nil
}

// returns true/false is user is ratelimited and when the ratelimit is lifted
func getUserMovieRatelimit(db dbhandle, userid int) (*time.Time, error) {
    var d *time.Time
    
    if err := db.QueryRow(context.Background(), SQL_USER_RATELIMIT, userid).Scan(&d); err != nil {
        return nil, err
    }
    
    return d, nil
}


//
// API
//

// generates token for api/account
func newApiToken(db dbhandle, userid int) (string, error) {
    var exists bool
    var secret string
    
    for {
        secret = randAsciiString(72)
        if err := db.QueryRow(context.Background(), SQL_APITOKEN_SECRET_EXISTS, secret).Scan(&exists); err != nil {
            return "", err
        }
        if !exists {
            break
        }
    }
    
    if _, err := db.Exec(context.Background(), SQL_APITOKEN_REGISTER, userid, secret); err != nil {
        return "", err
    }
    return secret, nil
}

func getApiTokenUserId(db dbhandle, secret string) (int, error) {
    var userid int

    if err := db.QueryRow(context.Background(), SQL_APITOKEN_GET_USERID, secret).Scan(&userid); err != nil {
        switch err {
        case pgx.ErrNoRows:
            return 0, nil
        default:
            return 0, err
        }
    }
    
    return userid, nil
}

// stub
// Invalidate token prematurely
func destroyApiToken(db dbhandle, secret string) error {
    return nil
}