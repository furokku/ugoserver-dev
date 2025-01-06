package main

import (
	"io"
	"net/http"
	"net/url"

	"time"
)

// we do not use nas much
// so this can be garbage
const (
    NAS_TOKEN string = "NDSflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocfloc"
    MELONDS_BSSID string = "00f077777777"
)

// dsi mode auth middleware
// check_id false : check for sid after basic authentication
// check_id true : check for userid after login
func dsi_am(check_id bool, next http.HandlerFunc) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        sid := r.Header.Get("X-Dsi-Sid")
        if err := isSidValid(sid); err != nil {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        
        if check_id && !sessions[sid].is_logged_in {
            w.WriteHeader(http.StatusUnauthorized)
            return
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
        w.Header()["X-DSi-SID"] = []string{generateUniqueSession()}

    case "POST":

        sid := r.Header.Get("X-Dsi-Sid")

        // fill out with initial data
        req := session{
            mac:      r.Header.Get("X-Dsi-Mac"),
            fsid:     r.Header.Get("X-Dsi-Id"),
            auth:     r.Header.Get("X-Dsi-Auth-Response"),
            sid:      sid,
            ver:      r.Header.Get("X-Ugomemo-Version"),
            username: r.Header.Get("X-Dsi-User-Name"),
            region:   r.Header.Get("X-Dsi-Region"),
            lang:     r.Header.Get("X-Dsi-Lang"),
            country:  r.Header.Get("X-Dsi-Country"),
            birthday: r.Header.Get("X-Birthday"),
            datetime: r.Header.Get("X-Dsi-Datetime"),
            color:    r.Header.Get("X-Dsi-Color"),
            
            ip: ip,
            issued: time.Now(),
        }

        ref := sid[:6] + "_" + req.mac[6:]

        if r, err := req.validate(); err != nil {
            // funkster detected
            infolog.Printf("%v did not pass auth validation (%v), ref %v", ip, err, ref)
            msg := "an error occured. try again later\nreference: " + ref

            if err == ErrFsidBan || err == ErrIpBan {
                msg = "you have been banned until\n" + r.expires.UTC().Format(time.DateTime) + " UTC"  + "\n\nreason: " + r.message + "\n\nreference: " + ref
            }

            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE(msg))
            return
        }

        // possible to add X-DSi-New/Unread-Notices here
        // for flashing NEW on inbox button
        w.Header()["X-DSi-SID"] = []string{sid}

        // fun part: figure out if user has registered before
        // whether logging in from the same ip
        // and obtain a user id
        userid, last_login_ip, err := getUserDsi(req.fsid)
        if err == ErrNoUser {
            req.is_unregistered = true
        } else if err != nil {
            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE("an error occured. try again later\nreference: " + ref))
            return
        }

        req.userid = userid
        if ip == last_login_ip {
            req.is_logged_in = true
        }
        
        sessions[sid] = req
        debuglog.Printf("new session %v : %v\n", sid, sessions[req.sid])
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

            if bssid == MELONDS_BSSID { // 00:F0:77 mac is unassigned. 100% emulator
                if err := issueBan("auto", time.Now().Add(60 * time.Minute), ip, "emulator [bssid]", "emulator usage", true); err == ErrAlreadyBanned {
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
                    resp.Set("token", nasEncode([]byte(NAS_TOKEN)))

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

func (a session) validate() (restriction, error) {

    // empty restriction
    e := restriction{}

    if ok, err := whitelistQueryFsid(a.fsid); err != nil {
        return e, err
    } else if ok {
        return e, nil
    }

    if b, r, err := queryBan(a.fsid); err != nil {
        return e, err
    } else if b {
        return r, ErrFsidBan
    }
    if b, r, err := queryBan(a.ip); err != nil {
        return e, err
    } else if b {
        return r, ErrIpBan
    }

    if a.mac[5:] != a.fsid[9:] {
        return e, ErrAuthMacFsidMismatch
    }
    if a.fsid[9:] == "BF112233" {
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

// issue a unique sid to the client
func generateUniqueSession() string {
    var sid string

    for {
        sid = randAsciiString(32)
        if _, ok := sessions[sid]; !ok {
            return sid
        }
    }
}

// find expired sessions and delete them every
// 5 minutes
func pruneSids() {
    for {
        time.Sleep(5 * time.Minute)

        for k, v := range sessions {
            if time.Since(v.issued).Seconds() >= 7200 {
                delete(sessions, k)
            }
        }
    }
}