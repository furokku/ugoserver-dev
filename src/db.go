package main

import (
    "time"
    "fmt"
    "database/sql"
)


// pass a statement prepared with db.Prepare and return flipnotes
func queryDbFlipnotes(stmt *sql.Stmt, args ...any) ([]flipnote) {

    var resp []flipnote

    rows, err := stmt.Query(args...)
    if err != nil {
        errorlog.Printf("failed to query database for flipnotes: %v", err)
        return []flipnote{}
    }

    defer rows.Close()

    for rows.Next() {
        r := flipnote{}

        // may remove parent author name/id/filename,
        // as they are basically never queried and can be pulled
        // from file if needed.
        rows.Scan(&r.id, &r.author_id, &r.author_name, &r.parent_author_id, &r.parent_author_name, &r.author_filename, &r.uploaded_at, &r.views, &r.downloads, &r.stars["yellow"], &r.stars["green"], &r.stars["red"], &r.stars["blue"], &r.stars["purple"], &r.lock, &r.deleted)
        resp = append(resp, r)
    }

    return resp
}

func updateViewDlCount(id int, t string) {
    var set string
    switch t {
    case "dl":
        set = "downloads"
    case "ppm":
        set = "views"
    }
    if _, err := db.Exec(fmt.Sprintf("UPDATE flipnotes SET %s = %s + 1 WHERE id = $1", set, set), id); err != nil {
        errorlog.Printf("%v", err)
    }
}

func deleteFlipnote(id int) {
    if _, err := db.Exec("UPDATE flipnotes SET deleted = true WHERE id = $1", id); err != nil {
        errorlog.Printf("%v", err)
    }
}

func updateStarCount(id int, color string, n int) {
    if _, err := db.Exec(fmt.Sprintf("UPDATE flipnotes SET %s_stars = %s_stars + %d WHERE id = $1", color, color, n), id); err != nil {
        errorlog.Printf("%v", err)
    }
}

func updateUserStarredMovies(id int, fsid string) {
}

func getFrontFlipnotes(ptype string, p int) ([]flipnote, int) {
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
        errorlog.Printf("error preparing statement %v", err)
        return []flipnote{}, 0
    }

    stmt2, err := db.Prepare("SELECT count(1) FROM flipnotes WHERE deleted = false")
    if err != nil {
        errorlog.Printf("error preparing statement %v", err)
        return []flipnote{}, 0
    }

    if err := stmt2.QueryRow().Scan(&total); err != nil {
        errorlog.Print(err)
        return []flipnote{}, 0
    }

    return queryDbFlipnotes(stmt1, offset), total
}

func getFlipnoteById(id int) flipnote {
    stmt, err := db.Prepare("SELECT * FROM flipnotes WHERE deleted = false AND id = $1")
    if err != nil {
        errorlog.Printf("%v", err)
        return flipnote{}
    }

    return queryDbFlipnotes(stmt, id)[0]
}

func checkFlipnoteExists(fn string) bool {
    var n int

    err := db.QueryRow("SELECT count(1) FROM flipnotes WHERE author_filename = $1", fn).Scan(&n)
    if err != nil {
        errorlog.Printf("could not check if flipnote %v exists: %v", fn, err)
        return false
    }

    if n != 0 {
        return true
    }

    return false
}

func whitelistAddId(id string) {
    if _, err := db.Exec("INSERT INTO auth_whitelist (fsid) VALUES ($1)", id); err != nil {
        errorlog.Printf("failed to whitelist %v: %v", id, err)
    }
}

func whitelistDelId(id string) {
    if _, err := db.Exec("DELETE FROM auth_whitelist WHERE fsid = $1", id); err != nil {
        errorlog.Printf("failed to unwhitelist %v: %v", id, err)
    }
}

func whitelistCheckId(id string) bool {
    var i int
    err := db.QueryRow("SELECT id FROM auth_whitelist WHERE fsid = $1", id).Scan(&i)
    if err == sql.ErrNoRows {
        return false
    } else if err != nil {
        errorlog.Printf("failed to query whitelist for %v: %v", id, err)
        return false
    }
    return true
}

func queryIsBanned(ident string) (bool, restriction) {
    b := restriction{}
    err := db.QueryRow("SELECT * FROM bans WHERE pardon = false AND affected = $1 AND expires > now() ORDER BY expires DESC LIMIT 1", ident).Scan(&b.id, &b.issuer, &b.issued, &b.expires, &b.reason, &b.message, &b.pardon, &b.affected)
    if err == sql.ErrNoRows {
        return false, restriction{}
    } else if err != nil {
        errorlog.Printf("failed to query ban for %v: %v", ident, err)
        return false, restriction{}
    }
    return true, b
}

func issueBan(iss string, exp time.Time, ident string, r string, msg string, ce bool) {
    if ce {
        if b, _ := queryIsBanned(ident); b {
            infolog.Printf("%v is already banned, ignoring", ident)
            return
        }
    }

    if _, err := db.Exec("INSERT INTO bans (issuer, expires, reason, message, affected) VALUES ($1, $2, $3, $4, $5)", iss, exp, r, msg, ident); err != nil {
        errorlog.Printf("failed to issue ban: %v", err)
    }
    infolog.Printf("%v banned %v until %v for %v (%v)", iss, ident, exp, r, msg)
}
