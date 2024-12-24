package main

import (
	"database/sql"
	"fmt"
	"time"
)

func connect() (*sql.DB, error) {
    var cs string

    switch cnf.DB.Type {
    case "sqlite3":
        cs = fmt.Sprintf("file:%s/%s?cache=shared&mode=rwc", cnf.Dir, cnf.DB.File)
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

// pass a statement prepared with db.Prepare and return flipnotes
func queryDbFlipnotes(stmt *sql.Stmt, args ...any) ([]flipnote, error) {

    var resp []flipnote

    rows, err := stmt.Query(args...)
    if err != nil {
        errorlog.Printf("failed to query database for flipnotes: %v", err)
        return []flipnote{}, err
    }

    defer rows.Close()

    for rows.Next() {
        r := flipnote{stars:make(map[string]int)}

        // may remove parent author name/id/filename,
        // as they are basically never queried and can be pulled
        // from file if needed.
        var y,g,tr,b,p int
        rows.Scan(&r.id, &r.author_id, &r.author_name, &r.parent_author_id, &r.parent_author_name, &r.author_filename, &r.uploaded_at, &r.views, &r.downloads, &y, &g, &tr, &b, &p, &r.lock, &r.deleted)
        r.stars["yellow"]=y
        r.stars["green"]=g
        r.stars["red"]=tr
        r.stars["blue"]=b
        r.stars["purple"]=p 
        resp = append(resp, r)
    }

    return resp, nil
}

func updateViewDlCount(id int, t string) error {
    var set string
    switch t {
    case "dl":
        set = "downloads"
    case "ppm":
        set = "views"
    }
    if _, err := db.Exec(fmt.Sprintf("UPDATE flipnotes SET %s = %s + 1 WHERE id = $1", set, set), id); err != nil {
        return err
    }
    return nil
}

func deleteFlipnote(id int) error {
    if _, err := db.Exec("UPDATE flipnotes SET deleted = true WHERE id = $1", id); err != nil {
        return err
    }
    return nil
}

func updateStarCount(id int, color string, n int) error {
    if _, err := db.Exec(fmt.Sprintf("UPDATE flipnotes SET %s_stars = %s_stars + %d WHERE id = $1", color, color, n), id); err != nil {
        return err
    }
    return nil
}

func updateUserStarredMovies(id int, fsid string) error {
    //todo
    return nil
}

func getFrontFlipnotes(ptype string, p int) ([]flipnote, int, error) {
    var orderby string
    var total int
    offset := findOffset(p)

    switch ptype {
    case "recent":
        orderby = "id"
    default:
        orderby = "id"
    }

    stmt1, err := db.Prepare(fmt.Sprintf("SELECT * FROM flipnotes WHERE deleted = false ORDER BY %s DESC LIMIT 50 OFFSET $1", orderby))
    if err != nil {
        return []flipnote{}, 0, err
    }

    stmt2, err := db.Prepare("SELECT count(1) FROM flipnotes WHERE deleted = false")
    if err != nil {
        return []flipnote{}, 0, err
    }

    if err := stmt2.QueryRow().Scan(&total); err != nil {
        return []flipnote{}, 0, err
    }

    flips, err := queryDbFlipnotes(stmt1, offset)
    if err != nil {
        return []flipnote{}, 0, err
    }

    return flips, total, nil
}

func getFlipnoteById(id int) (flipnote, error) {
    stmt, err := db.Prepare("SELECT * FROM flipnotes WHERE deleted = false AND id = $1")
    if err != nil {
        return flipnote{}, err
    }

    flip, err := queryDbFlipnotes(stmt, id)
    if err != nil {
        return flipnote{}, err
    }

    return flip[0], nil
}

func checkMovieExistsAfn(fn string) (bool, error) {
    var n int

    err := db.QueryRow("SELECT count(1) FROM flipnotes WHERE author_filename = $1", fn).Scan(&n)
    if err != nil {
        return false, err
    }

    if n != 0 {
        return true, nil
    }

    return false, nil
}

func whitelistAddId(id string) error {
    if _, err := db.Exec("INSERT INTO auth_whitelist (fsid) VALUES ($1)", id); err != nil {
        return err
    }
    return nil
}

func whitelistDelId(id string) error {
    if _, err := db.Exec("DELETE FROM auth_whitelist WHERE fsid = $1", id); err != nil {
        return err
    }
    return nil
}

func whitelistCheckId(id string) (bool, error) {
    var i int
    err := db.QueryRow("SELECT id FROM auth_whitelist WHERE fsid = $1", id).Scan(&i)
    if err == sql.ErrNoRows {
        return false, nil
    } else if err != nil {
        errorlog.Printf("failed to check whitelist (%v): %v", id, err)
        return false, err
    }
    return true, nil
}

func queryIsBanned(ident string) (bool, restriction, error) {
    b := restriction{}
    err := db.QueryRow("SELECT * FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1", ident).Scan(&b.id, &b.issuer, &b.issued, &b.expires, &b.reason, &b.message, &b.pardon, &b.affected)
    if err == sql.ErrNoRows {
        return false, restriction{}, nil
    } else if err != nil {
        return false, restriction{}, err
    }
    return true, b, nil
}

func issueBan(iss string, exp time.Time, ident string, r string, msg string, ce bool) error {
    if ce {
        if b, _, _ := queryIsBanned(ident); b {
            return ErrAlreadyBanned
        }
    }

    if _, err := db.Exec("INSERT INTO bans (issuer, expires, reason, message, affected) VALUES ($1, $2, $3, $4, $5)", iss, exp, r, msg, ident); err != nil {
        return err
    }
    infolog.Printf("%v banned %v until %v for %v (%v)", iss, ident, exp, r, msg)
    return nil
}
