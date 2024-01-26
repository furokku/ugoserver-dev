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

    log.Printf("%v made %v request to %v%v with header %v\n", r.Header.Get("X-Real-Ip"), r.Method, r.Host, r.URL.Path, r.Header)

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
            auth:     r.Header.Get("X-Dsi-Auth-Response"), // TODO: check this
            sid:      r.Header.Get("X-Dsi-Sid"),
            ver:      r.Header.Get("X-Ugomemo-Version"), // maybe only accept V2: done, mux regex does same thing
            username: r.Header.Get("X-Dsi-User-Name"),   // TODO: store this: done
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
            sessions[req.sid] = struct{
                fsid string
                username string
                issued int64
            } {
                fsid: req.id,
                username: decReqUsername(req.username),
                issued: time.Now().Unix(),
            }

            w.Header()["X-DSi-SID"] = []string{req.sid}

            // TODO: handle on per user basis
            // both of these do the same thing probably but
            // for convenience sake likely only one
            // will be set
            w.Header()["X-DSi-New-Notices"] = []string{"0"}
            w.Header()["X-DSi-Unread-Notices"] = []string{"0"}

            log.Printf("successfully authenticated new session %v: %v\n", req.sid, sessions[req.sid])
//          log.Println(sessions)
        }

    // technically no longer needed
    // but I'll keep it just coz
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    w.WriteHeader(http.StatusOK)
    log.Printf("responded to %v's request for %v%v with %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, w.Header())
}

func nasAuth(w http.ResponseWriter, r *http.Request) {

    // deny requests other than POST
    // this IS necessary because of requests being handled by
    // http.DefaultServeMux and not gorilla mux
    if r.Method != "POST" {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    body, _ := io.ReadAll(r.Body)
    nasRequest, err := url.ParseQuery(string(body))
    if err != nil {
        log.Printf("error parsing urlencoded form from %v at %v%v: %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, err)
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // decode base64 values to plaintext for logging reasons
    // and to check action key
    for key := range nasRequest {
        // only one value is set per key so this is fine
        nasRequest[key][0] = nasDecode(nasRequest[key][0])
    }

    // the form itself doesn't really convery much helpful information
    // from a logging standpoint, but you can just add in string(body)
    // here if you wish to see it
    // might add a config option for that later when that exists
    log.Printf("%v requested %v%v with headers %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, r.Header)

    action := nasRequest.Get("action")
    resp := make(url.Values)

    switch r.URL.Path {
    case "/ac":
        switch action {

        // known action values are login, acctcreate and svcloc
        // those can be handled later
        case "login":
            resp.Set("challenge", nasEncode(randAsciiString(8)))
            resp.Set("locator", nasEncode("gamespy.com"))
            resp.Set("retry", nasEncode("0"))
            resp.Set("returncd", nasEncode("001"))
            resp.Set("token", nasEncode(append([]byte("NDS"), randBytes(96)...)))

        default:
            w.WriteHeader(http.StatusBadRequest) // unimplemented functionality or something fishy
            return
        }

    // nintendo profanity filter thing
    case "/pr":
        // I don't really care about profanity but
        // a simple check could be added here for completeness
        resp.Set("prwords", nasEncode("0"))
        resp.Set("returncd", nasEncode("000"))

    default:
        w.WriteHeader(http.StatusNotFound) // invalid endpoint
        return
    }

    // datetime will be sent regardless
    resp.Set("datetime", nasEncode(time.Now().Format("20060102150405")))
    w.Write([]byte(resp.Encode()))
}
