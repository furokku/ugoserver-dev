package main


import (
    "io"
    "net/http"
    "net/url"

    "time"
)

var nastoken string = "NDSflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocfloc"


func hatenaAuth(w http.ResponseWriter, r *http.Request) {

    switch r.Method {

    // > only GET and POST requests will
    // > ever be sent
    // initial rev of flipnote studio sends
    // two GET requests, will need to consider later
    // how to handle that
    //
    // Likely wont
    case "GET":

        // pointless atm
        w.Header()["X-DSi-Unread-Notices"] = []string{"0"}
        w.Header()["X-DSi-New-Notices"] = []string{"0"}

        // TODO: validate auth challenge
        // something to do with XOR keys
        // is it really needed? probably not
        w.Header()["X-DSi-Auth-Challenge"] = []string{"mangoloco"}
        w.Header()["X-DSi-SID"] = []string{genUniqueSession()}

    case "POST":

        req := AuthPostRequest{
            mac:      r.Header.Get("X-Dsi-Mac"), //console mac
            id:       r.Header.Get("X-Dsi-Id"), //fsid
            auth:     r.Header.Get("X-Dsi-Auth-Response"),
            sid:      r.Header.Get("X-Dsi-Sid"),
            ver:      r.Header.Get("X-Ugomemo-Version"),
            username: r.Header.Get("X-Dsi-User-Name"),
            region:   r.Header.Get("X-Dsi-Region"),
            lang:     r.Header.Get("X-Dsi-Lang"),
            country:  r.Header.Get("X-Dsi-Country"),
            birthday: r.Header.Get("X-Birthday"),
            datetime: r.Header.Get("X-Dsi-Datetime"),
            color:    r.Header.Get("X-Dsi-Color"),
        }

        // TODO: function to validate auth request
//      if !req.validate() {
        if false {
            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE("eat concrete"))
            return
        } else {
            sessions[req.sid] = session{
                fsid: req.id,
                username: decReqUsername(req.username),
                issued: time.Now(),
                ip: r.Header.Get("X-Real-Ip"),
            }

            w.Header()["X-DSi-SID"] = []string{req.sid}

            // TODO: handle on per user basis
            // both of these do the same thing probably but
            // for convenience sake likely only one
            // will be set
            w.Header()["X-DSi-New-Notices"] = []string{"0"}
            w.Header()["X-DSi-Unread-Notices"] = []string{"0"}

            debuglog.Printf("new session %v : %v\n", req.sid, sessions[req.sid])
//          log.Println(sessions)
        }
    }

    w.WriteHeader(http.StatusOK)
}

func nasAuth(w http.ResponseWriter, r *http.Request) {

    body, _ := io.ReadAll(r.Body)
    nasRequest, err := url.ParseQuery(string(body))
    if err != nil {
        errorlog.Printf("bad form from %v at %v%v: %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, err)
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

    resp := make(url.Values)

    switch r.URL.Path {
        case "/ac":

            action := nasRequest.Get("action")
            switch action {

                // known action values are login, acctcreate and svcloc
                // those can be handled later
                case "login":
                    resp.Set("challenge", nasEncode(randAsciiString(8)))
                    resp.Set("locator", nasEncode("gamespy.com"))
                    resp.Set("retry", nasEncode("0"))
                    resp.Set("returncd", nasEncode("001"))
                    resp.Set("token", nasEncode([]byte(nastoken)))

                case "acctcreate":
                    resp.Set("retry", nasEncode("0"))
                    resp.Set("returncd", nasEncode("002"))
                    resp.Set("userid", nasEncode("notimportant"))

                default:
                    debuglog.Printf("action %s", action)
                    w.WriteHeader(http.StatusBadRequest) // unimplemented functionality or something fishy
                    return
            }

        // nintendo profanity filter thing
        case "/pr":
            // I don't really care about profanity but
            // a simple check could be added here for completeness
            resp.Set("prwords", nasEncode("0"))
            resp.Set("returncd", nasEncode("000"))
    }

    // datetime will be sent regardless
    resp.Set("datetime", nasEncode(time.Now().Format("20060102150405")))
    w.Write([]byte(resp.Encode()))
}
