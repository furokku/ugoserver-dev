package main

import (
    "database/sql"
    "time"
    "log"
)


// Fetch the latest uploaded flipnotes.
// 54 flipnotes are fetched, if so many exist per given offset
// Only really 53 are shown, but the 54th is just to determine whether to
// show the next page button
func getLatestFlipnotes(db *sql.DB, p int) ([]flipnote, int) {

    var query []flipnote
    var total int

    // find offset by page number
    offset := findOffset(p)

    rows, err := db.Query("SELECT * FROM flipnotes ORDER BY uploaded_at DESC LIMIT 54 OFFSET $1", offset)
    if err != nil {
        log.Fatalf("fetchLatestFlipnotes: %v", err)
    }

    // get amount of total flipnotes for relevant query
    rows2, err := db.Query("SELECT count(1) FROM flipnotes")
    if err != nil {
        log.Fatalf("fetchLatestFlipnotes: %v", err)
    }

    defer rows.Close()
    defer rows2.Close()

    for rows.Next() {
        var id int
        var author, filename string
        var uploaded_at time.Time

        rows.Scan(&id, &author, &filename, &uploaded_at)
        query = append(query, flipnote{id:id, author:author, filename:filename, uploaded_at:uploaded_at})
    }

    // this returns only one row, so this is fine
    // a failsafe should not be needed
    rows2.Next()
    rows2.Scan(&total)

    return query, total
}
