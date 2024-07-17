package main

import (
    "time"
)


// Fetch the latest uploaded flipnotes.
// > 54 flipnotes are fetched, if so many exist per given offset
// > Only really 53 are shown, but the 54th is just to determine whether to
// > show the next page button
//
// Probably terrible practice so 50 flipnotes are requested
// and the total amount is too in order to build a page count
// and determine whether the next page button should be there
func getFrontFlipnotes(q string, p int) ([]flipnote, int) {

    var resp []flipnote
    var total int
    var orderby string

    // find offset by page number
    offset := findOffset(p)
    switch q {
    case "recent":
        orderby = "id"
    default:
        orderby = "id"
    }

    rows, err := db.Query("SELECT * FROM flipnotes ORDER BY $1 ASC LIMIT 50 OFFSET $2", orderby, offset)
    if err != nil {
        // TODO: return an error for this and below and other stuff
        errorlog.Printf("failed to access database: %v", err)
        return []flipnote{}, 0
    }

    // get amount of total flipnotes in order to do some math
    rows2, err := db.Query("SELECT count(1) FROM flipnotes")
    if err != nil {
        errorlog.Printf("failed to access database: %v", err)
        return []flipnote{}, 0
    }

    defer rows.Close()
    defer rows2.Close()

    for rows.Next() {
        var id, l int
        var aid, an, paid, pan, afn string
        var u time.Time

        rows.Scan(&id, &aid, &an, &paid, &pan, &afn, &u, &l)
        resp = append(resp, flipnote{id:id, author_id:aid, author_name:an, parent_author_id:paid, parent_author_name:pan, author_filename:afn, uploaded_at:u, lock:l})
    }

    // this returns only one row, so this is fine
    // a failsafe should not be needed
    rows2.Next()
    rows2.Scan(&total)

    return resp, total
}
