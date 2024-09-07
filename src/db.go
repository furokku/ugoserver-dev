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
        errorlog.Printf("failed to access database: %v", err)
        return []flipnote{}
    }

    defer rows.Close()

    for rows.Next() {
        //mess
        var id, v, dl, ys, gs, rs, bs, ps int
        var aid, an, paid, pan, afn string
        var l, del bool
        var u time.Time

        rows.Scan(&id, &aid, &an, &paid, &pan, &afn, &u, &v, &dl, &ys, &gs, &rs, &bs, &ps, &l, &del)
        resp = append(resp, flipnote{id:id, author_id:aid, author_name:an, parent_author_id:paid, parent_author_name:pan, author_filename:afn, uploaded_at:u, lock:l, views:v, downloads:dl, stars:map[string]int{"yellow":ys,"green":gs,"red":rs,"blue":bs,"purple":ps}, deleted:del})
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
