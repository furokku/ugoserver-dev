package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"time"

	"strconv"
)

const (
    // nas isn't used here so this can be garbage
    NAS_TOKEN string = "NDSflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocfloc"
    MELONDS_BSSID string = "00f077777777" // default melonAP bssid, catches emulators

    MSG_NO_SUPPORT string = "you are using an outdated version of flipnote studio.\nplease connect using the latest version."
    MSG_EARLY_ERROR string = "an error occurred during\nearly authentication."
    MSG_ERROR_REF string = "an error occured. try again later\nreference: "
)

// dsi_am middleware checks whether a user is logged in, and optionally redirects them
// to a log in page if they are not
func dsi_am(next http.HandlerFunc, check_id bool, redirect bool) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        sid := r.Header.Get("X-Dsi-Sid")
        s := sessions[sid]
        // Not authenticated thru flipnote
        if err := isSidValid(sid); err != nil {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        
        if check_id && !s.IsLoggedIn {
            if redirect {
                http.HandlerFunc(sa).ServeHTTP(w, r)
                return
            }
            w.WriteHeader(http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    }
    
    return fn
}

// hatenaAuth handler authenticates clients after NAS on /ds/v2-xx/auth
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

        region, err := strconv.Atoi(r.Header.Get("X-Dsi-Region"))
        if err != nil {
            // possible rev2
            region = 0
        //    errorlog.Printf("%s tried to authenticate with invalid region (%v)", ip, err)
        //    w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        //    w.Write(encUTF16LE(MSG_EARLY_ERROR))
        //    return
        }

        ver, err := strconv.Atoi(r.Header.Get("X-Ugomemo-Version"))
        if err != nil {
            errorlog.Printf("%s tried to authenticate with invalid version (%v)", ip, err)
            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE(MSG_EARLY_ERROR))
            return
        }

        // fill out with initial data
        req := Session{
            MAC:      r.Header.Get("X-Dsi-Mac"),
            FSID:     r.Header.Get("X-Dsi-Id"),
            Auth:     r.Header.Get("X-Dsi-Auth-Response"),
            Ver:      ver,
            Username: qd(r.Header.Get("X-Dsi-User-Name")),
            Region:   region, // unset on rev2
            Lang:     r.Header.Get("X-Dsi-Lang"), // unset on rev2
            Country:  r.Header.Get("X-Dsi-Country"), // unset on rev2
            Birthday: r.Header.Get("X-Birthday"),
            DateTime: r.Header.Get("X-Dsi-Datetime"), // unset on rev2
            Color:    r.Header.Get("X-Dsi-Color"), // unset on rev2
            
            IP: ip,
            Issued: time.Now(),
        }

        ref := sid[:6] + "_" + req.MAC[6:]

        if r, err := req.validate(); err != nil {
            // funkster detected
            infolog.Printf("%v did not pass auth validation (%v), ref %v", ip, err, ref)
            msg := MSG_ERROR_REF + ref

            if err == ErrFsidBan || err == ErrIpBan {
                msg = fmt.Sprintf("you have been banned until\n%s UTC\n\nreason: %s\n\nreference: %s", r.Expires.UTC().Format(time.DateTime), r.Message, ref)
            }

            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE(msg))
            return
        }

        // figure out if user has registered before,
        // whether logging in from the same ip
        // and obtain a user id
        UserID, last_login_ip, err := getUserDsi(req.FSID)
        if err == ErrNoUser {
            req.IsUnregistered = true
        } else if err != nil {
            errorlog.Printf("while getting user: %v", err)
            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE(MSG_ERROR_REF + ref))
            return
        }

        req.UserID = UserID
        if ip == last_login_ip {
            req.IsLoggedIn = true
        }
        
        sessions[sid] = req
        //debuglog.Printf("new session %v : %v\n", sid, sessions[sid])

        // if nothing else failed, set SID header
        // possible to add X-DSi-New/Unread-Notices here
        // for flashing NEW on inbox button
        w.Header()["X-DSi-SID"] = []string{sid}
    }

    w.WriteHeader(http.StatusOK)
}

// nosupport handler accept rev2 clients on /ds/v2/auth and rejects them with a message
func nosupport(w http.ResponseWriter, r *http.Request) {

    switch r.Method {

    case "GET":
        w.Header()["X-DSi-SID"] = []string{"x"}
        w.Header()["X-DSi-Auth-Challenge"] = []string{"boyfantasy"}
        w.WriteHeader(http.StatusOK)

    case "POST":
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE(MSG_NO_SUPPORT))
    }
}

// nasAuth handler accepts queries to /ac (nas account) and /pr (nas profanity check);
// requests here are logged, but don't provide much useful information other than BSSID, which
// is used to determine if a client is an emulator. This only works if the emulator's AP BSSID has
// not been changed, however, which it usually isn't
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
                if err := issueBan("auto", time.Now().Add(60 * time.Minute), ip, "emulator usage", true); err == ErrAlreadyBanned {
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
                    resp.Set("UserID", nasEncode("notimportant"))

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

// validate() method will take a session and check if the user is banned or otherwise has any inconsistencies
func (a Session) validate() (Ban, error) {

    // empty restriction
    e := Ban{}


    if ok, err := whitelistQueryFsid(a.FSID); err != nil {
        return e, err
    } else if ok {
        return e, nil
    }

    if b, r, err := queryBan(a.FSID); err != nil {
        return e, err
    } else if b {
        return r, ErrFsidBan
    }
    if b, r, err := queryBan(a.IP); err != nil {
        return e, err
    } else if b {
        return r, ErrIpBan
    }

    if a.MAC[5:] != a.FSID[9:] {
        return e, ErrAuthMacFsidMismatch
    }
    if a.FSID[9:] == "BF112233" {
        return e, ErrAuthEmulatorId
    }
    if a.MAC == "0009BF112233" {
        return e, ErrAuthEmulatorMac
    }
    if a.age() < 13 {
        return e, ErrAuthUnderage
    }

    return e, nil
}

// isSidValid() checks if a sid is taken
func isSidValid(sid string) error {
    if _, ok := sessions[sid]; ok {
        return nil
    }
    return ErrNoSid
}

// genSid() returns a new, unusued session identifier
func generateUniqueSession() string {
    var sid string

    for {
        sid = randAsciiString(32)
        if _, ok := sessions[sid]; !ok {
            return sid
        }
    }
}

// pruneSids() will run indefinitely and, every 5 minutes, loop through the map of sessions
// and remove the expired sessions
func pruneSids() {
    for {
        time.Sleep(5 * time.Minute)

        for k, v := range sessions {
            if time.Since(v.Issued).Seconds() >= 1800 {
                delete(sessions, k)
            }
        }
    }
}

// getregion() returns a two letter region code for urls
func (s Session) getregion() string {
    var r string
    switch s.Region {
    case 0:
        r = "jp"
    case 1:
        r = "us"
    case 2:
        r = "eu"
    }
    return r
}

// sa handler returns a template for web based authentication
func sa(w http.ResponseWriter, r *http.Request) {
    s := sessions[r.Header.Get("X-Dsi-Sid")]
    ret := r.URL.Query().Get("ret")

    if err := templates.ExecuteTemplate(w, "auth.html", Page{
        Session: s,
        Root: cnf.Root,
        Region: s.getregion(),
        Return: ret,
    }); err != nil {
        errorlog.Printf("while executing template: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
    }
}

// sa_login_kbd handler takes the input from a keyboard POST and checks if the password is correct for the user
func sa_login_kbd(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := sessions[sid]
    in := r.Header.Get("X-Email-Addr")
    
    // sanity check
    if s.IsUnregistered || s.IsLoggedIn {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    if v, err := verifyUserDsi(s.UserID, in); err != nil {
        // handle error
        errorlog.Printf("while verifying user %d, %v", s.UserID, err)
        w.Header()["X-DSi-Forwarder"] = []string{s.ub("sa/auth.htm?ret=success")}
        return
    } else if !v {
        w.Header()["X-DSi-Forwarder"] = []string{s.ub("sa/auth.htm?ret=invalid")}
        return
    }

    if err := updateUserLastLogin(s.UserID, s.IP); err != nil {
        errorlog.Printf("while updating last login ip: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    s.IsLoggedIn = true
    sessions[sid] = s

    w.Header()["X-DSi-Forwarder"] = []string{s.ub("sa/auth.htm?ret=success")}
    w.WriteHeader(http.StatusOK)
}

// sa_reg_kbd handler takes the input from a keyboard POST and registers the user
func sa_reg_kbd(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := sessions[sid]
    in := r.Header.Get("X-Email-Addr")

    if !s.IsUnregistered { // user must be unregistered
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    id, err := registerUserDsi(s.Username, in, s.FSID, s.IP)
    if err != nil {
        debuglog.Printf("while registering user: %v", err)
        w.Header()["X-DSi-Forwarder"] = []string{s.ub("sa/auth.htm?ret=error")}
        return
    }

    s.UserID = id
    s.IsUnregistered = false
    s.IsLoggedIn = true

    sessions[sid] = s

    w.Header()["X-DSi-Forwarder"] = []string{s.ub("sa/auth.htm?ret=success")}
    w.WriteHeader(http.StatusOK)
}