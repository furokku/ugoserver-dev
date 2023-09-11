package main


import (
    "fmt"
    "log"

    "io"
    "net/http"
    "net/url"

    "time"
)


func hatenaAuth(w http.ResponseWriter, r *http.Request) {

    log.Printf("received %v request to %v%v with header %v\n", r.Method, r.Host, r.URL.Path, r.Header)

    // feels kinda redundant but i wrote this
    // earlier and don't feel like removing it (entirely)
//  match, _ := regexp.MatchString("/ds/v2(-[a-z]{2})?/auth", r.URL.Path)
//  vars := mux.Vars(r)

    // verify region in auth url is correct
    // mux inline regex works so this can be commented out
//  if !slices.Contains(regions, vars["reg"]) {
//      http.Error(w, "invalid region", http.StatusNotFound)
//      log.Printf("response 404 (invalid region) at %v%v", r.Host, r.URL.Path)
//      return
//  }

    switch r.Method {

    // > only GET and POST requests will
    // > ever be sent
    // correction: initial rev of flipnote studio sends
    // two GET requests, will need to consider later
    // how to handle that
    //
    // Likely wont
    case "GET":

        // seems like it's used to handle some sort of
        // server-wide notifications, as opposed to
        // user-specific ones which could be set later
        // TODO: Maybe this but seems unnecessary
        // Could be read from database if implemented
        const serverUnread int = 0

        if (serverUnread != 0) && (serverUnread != 1) {
            w.WriteHeader(http.StatusInternalServerError)
            return
        } else {
            w.Header()["X-DSi-Unread-Notices"] = []string{fmt.Sprint(serverUnread)}
            w.Header()["X-DSi-New-Notices"] = []string{fmt.Sprint(serverUnread)}
        }

        // TODO: validate auth challenge
        // I know it has something to do with XOR keys
        // but is it really needed? probably not
        w.Header()["X-DSi-Auth-Challenge"] = []string{randAsciiString(8)}
        w.Header()["X-DSi-SID"] = []string{genUniqueSession()}

    case "POST":

        req := authPostRequest{
            mac:      r.Header.Get("X-Dsi-Mac"),
            id:       r.Header.Get("X-Dsi-Id"),          // FSID
            auth:     r.Header.Get("X-Dsi-Auth-Response"),
            sid:      r.Header.Get("X-Dsi-Sid"),
            ver:      r.Header.Get("X-Ugomemo-Version"), // maybe only accept V2
            username: r.Header.Get("X-Dsi-User-Name"),   // TODO: store this
            region:   r.Header.Get("X-Dsi-Region"),
            lang:     r.Header.Get("X-Dsi-Lang"),
            country:  r.Header.Get("X-Dsi-Country"),
            birthday: r.Header.Get("X-Birthday"),        // weird how this one doesn't have DSi in it
            datetime: r.Header.Get("X-Dsi-Datetime"),
            color:    r.Header.Get("X-Dsi-Color"),
        }

        // TODO: function to validate auth request
//      if !req.validate() {
        if false {
            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE("error authenticating!"))
            return
        } else {
            sessions[req.sid] = struct{fsid string; issued int64}{fsid: req.id, issued: time.Now().Unix()}
            w.Header()["X-DSi-SID"] = []string{req.sid}

            // TODO: handle on per user basis
            // both of these do the same thing probably but
            // for convenience sake likely only one
            // will be set
            w.Header()["X-DSi-New-Notices"] = []string{"0"}
            w.Header()["X-DSi-Unread-Notices"] = []string{"0"}

//          log.Println(sessions)
        }

    // technically no longer needed
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        log.Printf("response 405 at %v%v", r.Host, r.URL.Path)
        return
    }

    w.WriteHeader(http.StatusOK)
    log.Printf("response 200 at %v%v with header %v\n", r.Host, r.URL.Path, w.Header())
}

func nasAuth(w http.ResponseWriter, r *http.Request) {

    // deny requests other than POST
    if r.Method != "POST" {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    body, _ := io.ReadAll(r.Body)
    nasRequest, err := url.ParseQuery(string(body))
    if err != nil {
        log.Fatal("error parsing urlencoded form")
    }

    // decode base64 values to plaintext for logging reasons
    // and to check action key
    for key := range nasRequest {
        // only one value is set per key so this is fine
        nasRequest[key][0] = decode(nasRequest[key][0])
    }

    log.Printf("received request to %v%v with data %v\n%v\n", r.Host, r.URL.Path, string(body), r.Header)
    log.Printf("%v\n\n", nasRequest)

    action := nasRequest.Get("action")
    resp := make(url.Values)

    switch r.URL.Path {
    case "/ac":
        switch action {

        // known action values are login, acctcreate and svcloc
        // those can be handled later
        case "login":
            resp.Set("challenge", encode(randAsciiString(8)))
            resp.Set("locator", encode("gamespy.com"))
            resp.Set("retry", encode("0"))
            resp.Set("returncd", encode("001"))
            resp.Set("token", encode(append([]byte("NDS"), randBytes(96)...)))

        default:
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

    // nintendo profanity filter thing
    case "/pr":
        resp.Set("prwords", encode("0"))
        resp.Set("returncd", encode("000"))

    default:
        http.Error(w, "invalid request", http.StatusNotFound)
        return
    }

    // datetime will be sent regardless
    resp.Set("datetime", encode(time.Now().Format("20060102150405")))
    w.Write([]byte(resp.Encode()))
}
