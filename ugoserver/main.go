package main

import (
    _ "github.com/lib/pq"
    "database/sql"
    "log"
)

func main() {

    // connect to database
    db, err := sql.Open("postgres", "postgresql://furokku:passwd@localhost/ugo?sslmode=disable")
    if err != nil {
        log.Fatalf("failed to open database (sql.Open): %v", err)
    } else {
        if err := db.Ping(); err != nil {
            log.Fatalf("failed to reach database (sql.Ping): %v", err)
        }
    }
    log.Println("database up")

    // start a thread to remove old, expired sessions
    // the time for a session to expire is 2 hours
    // may increase later if needed
    go pruneSessions()
    log.Println("session pruning up")

    // start the hatena auth/general http server
    //
    // ~~in future this may run on the main goroutine as
    // nas is not explicitly required thanks to wiimmfi
    // and such~~
    // will implement signal handling later so this should
    // still be a separate thread
    //
    // db should be passed because... db
    go runHatenaServer(db)
    log.Println("hatena server up")

    // need to choose whether to use own nas auth or to
    // use wiimmfi/kaeru nas
    // wiimmfi seems to be kinda weird and unstable because
    // it returns a 404 on /ac or /pr randomly
//  runNasServer()

    // hang main goroutine so that it doesn't
    // exit prematurely
    select {}
}
