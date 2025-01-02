package main

import (
	"io"
	"net/http"
	"net/url"

	"time"
)

const nastoken string = "NDSflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocfloc"

// middleware authorizer
// short name for convenience
func a(next http.HandlerFunc) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        if err := isSidValid(r.Header.Get("X-Dsi-Sid")); err != nil {
            w.WriteHeader(http.StatusUnauthorized)
        }

        next.ServeHTTP(w, r)
    }
    
    return fn
}

func hatenaAuth(w http.ResponseWriter, r *http.Request) {

    ip := r.Header.Get("X-Real-Ip")

    switch r.Method {

    // > only GET and POST requests will
    // > ever be sent
    // initial rev of flipnote studio sends
    // two GET requests, will need to consider later
    // how to handle that
    //
    // Likely wont
    case "GET":

        // something to do with XOR keys
        // is it really needed? probably not
        w.Header()["X-DSi-Auth-Challenge"] = []string{"mangoloco"}
        w.Header()["X-DSi-SID"] = []string{genUniqueSession()}

    case "POST":

        req := AuthPostRequest{
            mac:      r.Header.Get("X-Dsi-Mac"), //console mac
            id:       r.Header.Get("X-Dsi-Id"), //fsid
            ip:       ip,
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

        if r, err := req.validate(); err != nil {
            // funkster detected
            ref := req.sid[:6] + "_" + req.mac[6:]
            infolog.Printf("%v did not pass auth validation (%v), ref %v", ip, err, ref)
            msg := "an error occured. try again later\nreference: " + ref

            if err == ErrIdBan || err == ErrIpBan {
                msg = "you have been banned until\n" + r.expires.UTC().Format(time.DateTime) + " UTC"  + "\n\nreason: " + r.message + "\n\nreference: " + ref
            }

            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE(msg))
            return
        } else {
            sessions[req.sid] = session{
                fsid: req.id,
                ip: ip,
                issued: time.Now(),
                s2r: req, // store all other information upon authentication
            }

            w.Header()["X-DSi-SID"] = []string{req.sid}
            debuglog.Printf("new session %v : %v\n", req.sid, sessions[req.sid])
        }
//          log.Println(sessions)
    }

    w.WriteHeader(http.StatusOK)
}

func nasAuth(w http.ResponseWriter, r *http.Request) {

    ip := r.Header.Get("X-Real-Ip")
    body, _ := io.ReadAll(r.Body)
    nasRequest, err := url.ParseQuery(string(body))
    if err != nil {
        errorlog.Printf("bad nas form from %v: %v", ip, err)
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // decode base64 values to plaintext for logging reasons
    // and to check action key
    for key, val := range nasRequest {
        // only one value is set per key so this is fine
        dec, err := nasDecode(val[0])
        if err != nil {
            errorlog.Printf("error parsing NAS form key (value) %v (%v): %v", key, val, err)
            w.WriteHeader(http.StatusBadRequest)
            return
        }

        nasRequest[key][0] = dec
    }

    // the form itself doesn't really convery much helpful information
    // from a logging standpoint, but you can just add in string(body)
    // here if you wish to see it
    // might add a config option for that later when that exists

    resp := make(url.Values)

    switch r.URL.Path {
        case "/ac":

            bssid := nasRequest.Get("bssid")
            // Emulator check
            // Most users who try to use an emulator won't
            // change the default AP BSSID, which will give awau
            // the fact that they're using one

            if bssid == "00f077777777" { // 00:F0:77 mac is unassigned. 100% emulator
                err := issueBan("auto", time.Now().Add(60 * time.Minute), ip, "emulator [bssid]", "emulator usage", true)
                if err == ErrAlreadyBanned {
                    infolog.Printf("%v is already banned", ip)
                } else if err != nil {
                    errorlog.Printf("failed to issue ban for %v: %v", ip, err)
                }
            }

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

func (a AuthPostRequest) validate() (restriction, error) {

    // empty restriction
    e := restriction{}

    if ok, _ := whitelistCheckId(a.id); ok {
        return e, nil
    }

    if b, r, _ := queryIsBanned(a.id); b {
        return r, ErrIdBan
    }
    if b, r, _ := queryIsBanned(a.ip); b {
        return r, ErrIpBan
    }

    if a.mac[5:] != a.id[9:] {
        return e, ErrAuthMacIdMismatch
    }
    if a.id[9:] == "BF112233" {
        return e, ErrAuthEmulatorId
    }
    if a.mac == "0009BF112233" {
        return e, ErrAuthEmulatorMac
    }
    if age(a.birthday) < 13 {
        return e, ErrAuthUnderage
    }

    return e, nil
}

func isSidValid(sid string) error {
    if _, ok := sessions[sid]; ok {
        return nil
    }
    return ErrNoSid
}