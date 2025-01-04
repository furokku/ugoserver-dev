package main

import (
	"database/sql"
	"fmt"
	"time"
)

const (
    SQL_MOVIE_COUNT string = "SELECT count(1) FROM movies WHERE deleted = false"

    SQL_MOVIE_GET_ID string = "SELECT * FROM movies WHERE deleted = false AND id = $1"
    SQL_MOVIE_GET_RECENT string = "SELECT * FROM movies WHERE deleted = false ORDER BY uploaded_at DESC LIMIT 50 OFFSET $1"
    
    SQL_MOVIE_UPDATE_YELLOW_STAR string = "UPDATE movies SET yellow_stars = yellow_stars + $1 WHERE id = $2"
    SQL_MOVIE_UPDATE_GREEN_STAR string = "UPDATE movies SET green_stars = green_stars + $1 WHERE id = $2"
    SQL_MOVIE_UPDATE_RED_STAR string = "UPDATE movies SET red_stars = red_stars + $1 WHERE id = $2"
    SQL_MOVIE_UPDATE_BLUE_STAR string = "UPDATE movies SET blue_stars = blue_stars + $1 WHERE id = $2"
    SQL_MOVIE_UPDATE_PURPLE_STAR string = "UPDATE movies SET purple_stars = purple_stars + $1 WHERE id = $2"

    SQL_MOVIE_UPDATE_DL string = "UPDATE movies SET downloads = downloads + 1 WHERE id = $1"
    SQL_MOVIE_UPDATE_VIEWS string = "UPDATE movies SET views = views + 1 WHERE id = $1"
    
    SQL_MOVIE_ADD string = "INSERT INTO movies (author_id, author_name, parent_author_id, parent_author_name, author_filename, lock) VALUES ($1, $2, $3, $4, $5, $6) RETURNING (id)"
    SQL_MOVIE_DELETE string = "UPDATE movies SET deleted = true WHERE id = $1"
    
    SQL_MOVIE_CHECK_EXISTS_AFN string = "SELECT EXISTS(SELECT 1 FROM movies WHERE author_filename = $1) AS \"EXISTS\""
    
    SQL_WHITELIST_FSID_ADD string = "INSERT INTO auth_whitelist (fsid) VALUES ($1)"
    SQL_WHITELIST_FSID_DELETE string = "DELETE FROM auth_whitelist WHERE fsid = $1"
    SQL_WHITELIST_FSID_CHECK string = "SELECT EXISTS(SELECT 1 FROM auth_whitelist WHERE fsid = $1) AS \"EXISTS\""
    
    SQL_BAN_CHECK string = "SELECT EXISTS(SELECT 1 FROM bans WHERE affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1) AS \"EXISTS\""
    SQL_BAN_QUERY string = "SELECT * FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1"
    SQL_BAN_ISSUE string = "INSERT INTO bans (issuer, expires, reason, message, affected) VALUES ($1, $2, $3, $4, $5)"
    SQL_BAN_PARDON_ID string = "UPDATE bans SET pardon = true WHERE id = $1"
    
    SQL_USER_REGISTER string = "INSERT INTO users (username, password) VALUES ($1, crypt($2, gen_salt('bf')))"
    SQL_USER_VERIFY string = "SELECT id FROM users WHERE username = $1 AND password = crypt($2, password)"
    SQL_USER_CHECK_ADMIN string = "SELECT EXISTS(SELECT 1 FROM users WHERE admin = true AND id = $1) AS \"EXISTS\""

    SQL_APITOKEN_SECRET_EXISTS string = "SELECT EXISTS(SELECT 1 FROM apitokens WHERE expires > now() AND secret = crypt($1, secret)) AS \"EXISTS\""
    SQL_APITOKEN_REGISTER string = "INSERT INTO apitokens (userid, secret) VALUES ($1, crypt($2, gen_salt('bf')))"
    SQL_APITOKEN_VERIFY string = "SELECT users.id FROM apitokens JOIN users ON apitokens.userid = users.id WHERE expires > now() AND apitokens.secret = crypt($1, apitokens.secret)"
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
func queryDbMovies(stmt string, args ...any) ([]flipnote, error) {

    var resp []flipnote

    rows, err := db.Query(stmt, args...)
    if err != nil {
        return []flipnote{}, err
    }

    defer rows.Close()

    for rows.Next() {
        r := flipnote{stars:make(map[string]int)}

        // maybe replace this with root author? To show "original creator" perhaps
        var y,g,tr,b,p int
        if err := rows.Scan(&r.id, &r.author_id, &r.author_name, &r.parent_author_id, &r.parent_author_name, &r.author_filename, &r.uploaded_at, &r.views, &r.downloads, &y, &g, &tr, &b, &p, &r.lock, &r.deleted, &r.channel); err != nil {
            return []flipnote{}, err
        }
        r.stars["yellow"]=y
        r.stars["green"]=g
        r.stars["red"]=tr
        r.stars["blue"]=b
        r.stars["purple"]=p 
        resp = append(resp, r)
        fmt.Println("appended flipnote", r)
    }

    return resp, nil
}

func updateViewDlCount(id int, t string) error {
    var q string
    switch t {
    case "dl":
        q = SQL_MOVIE_UPDATE_DL
    case "ppm":
        q = SQL_MOVIE_UPDATE_VIEWS
    }
    if _, err := db.Exec(q, id); err != nil {
        return err
    }
    return nil
}

func addMovie(aid string, an string, paid string, pan string, afn string, l int) (int, error) {
    var id int

    // check if flipnote has already been uploaded
    // using filename (they are always unique)
    if exists, err := checkMovieExistsAfn(afn); err != nil {
        return 0, err
    } else if exists {
        return 0, ErrMovieExists
    }
    
    if err := db.QueryRow(SQL_MOVIE_ADD, aid, an, paid, pan, afn, l).Scan(&id); err != nil {
        return 0, err
    }
    
    return id, nil
}

func deleteMovie(id int) error {
    if _, err := db.Exec(SQL_MOVIE_DELETE, id); err != nil {
        return err
    }
    return nil
}

func updateMovieStars(id int, color string, n int) error {
    var q string
    switch color {
    case "yellow":
        q = SQL_MOVIE_UPDATE_YELLOW_STAR
    case "green":
        q = SQL_MOVIE_UPDATE_GREEN_STAR
    case "red":
        q = SQL_MOVIE_UPDATE_RED_STAR
    case "blue":
        q = SQL_MOVIE_UPDATE_BLUE_STAR
    case "purple":
        q = SQL_MOVIE_UPDATE_PURPLE_STAR
    }
    if _, err := db.Exec(q, n, id); err != nil {
        return err
    }
    return nil
}

func updateUserStarredMovies(id int, fsid string) error {
    //todo
    return nil
}

func getFrontMovies(ptype string, p int) ([]flipnote, int, error) {
    var total int
    var q string
    offset := findOffset(p)
    
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

    memos, err := queryDbMovies(q, offset)
    if err != nil {
        return []flipnote{}, 0, err
    }
    
    fmt.Println("returned flipnotes", memos)

    return memos, total, nil
}

func getMovieById(id int) (flipnote, error) {
    memo, err := queryDbMovies(SQL_MOVIE_GET_ID, id)
    if err != nil {
        return flipnote{}, err
    }

    return memo[0], nil
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
    var exists bool
    err := db.QueryRow(SQL_WHITELIST_FSID_CHECK, fsid).Scan(&exists)
    if err != nil {
        return false, err
    }
    return exists, nil
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
    err := db.QueryRow(SQL_BAN_QUERY, affected).Scan(&b.id, &b.issuer, &b.issued, &b.expires, &b.reason, &b.message, &b.pardon, &b.affected)
    if err == sql.ErrNoRows {
        return false, restriction{}, nil
    } else if err != nil {
        return false, restriction{}, err
    }
    return true, b, nil
}

func issueBan(iss string, exp time.Time, affected string, r string, msg string, ce bool) error {
    if ce {
        if b, _ := checkIsBanned(affected); b {
            return ErrAlreadyBanned
        }
    }

    if _, err := db.Exec(SQL_BAN_ISSUE, iss, exp, r, msg, affected); err != nil {
        return err
    }
    infolog.Printf("%v banned %v until %v for %v (%v)", iss, affected, exp, r, msg)
    return nil
}

func pardonBanId(id int) error {
    if _, err := db.Exec(SQL_BAN_PARDON_ID, id); err != nil {
        return err
    }
    return nil
}

func registerUser(username string, password string) error {
    if _, err := db.Exec(SQL_USER_REGISTER, username, password); err != nil {
        return err
    }
    return nil
}

func verifyUser(username string, password string) (int, error) {
    var id int
    if err := db.QueryRow(SQL_USER_VERIFY, username, password).Scan(&id); err != nil {
        return 0, err
    }
    return id, nil
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