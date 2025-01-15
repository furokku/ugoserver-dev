package main

import (
	"io"
	"net/http"
	"net/url"

	"time"

	"strconv"
)

const (
    // we do not use nas much
    // so this can be garbage
    NAS_TOKEN string = "NDSflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocfloc"
    MELONDS_BSSID string = "00f077777777"
    MSG_EARLY_ERROR string = "an error occurred during\nearly authentication."
    MSG_ERROR string = ""
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
        req := session{
            mac:      r.Header.Get("X-Dsi-Mac"),
            fsid:     r.Header.Get("X-Dsi-Id"),
            auth:     r.Header.Get("X-Dsi-Auth-Response"),
            ver:      ver,
            username: r.Header.Get("X-Dsi-User-Name"),
            region:   region, // unset on rev2
            lang:     r.Header.Get("X-Dsi-Lang"), // unset on rev2
            country:  r.Header.Get("X-Dsi-Country"), // unset on rev2
            birthday: r.Header.Get("X-Birthday"),
            datetime: r.Header.Get("X-Dsi-Datetime"), // unset on rev2
            color:    r.Header.Get("X-Dsi-Color"), // unset on rev2
            
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
        //debuglog.Printf("new session %v : %v\n", sid, sessions[sid])

        // if nothing else failed, set SID header
        // possible to add X-DSi-New/Unread-Notices here
        // for flashing NEW on inbox button
        w.Header()["X-DSi-SID"] = []string{sid}
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

func (s session) getregion() string {
    var r string
    switch s.region {
    case 0:
        r = "jp"
    case 1:
        r = "us"
    case 2:
        r = "eu"
    }
    return r
}

// sa: login page
func sa_login(w http.ResponseWriter, r *http.Request) {
    s := sessions[r.Header.Get("X-Dsi-Sid")]
    ret := r.URL.Query().Get("ret")

    // sanity check
    if s.is_logged_in || s.is_unregistered {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    switch ret {
    case "error":
        w.Write([]byte("<html><head></head><body>an error occurred</body></html>"))
        return
    case "invalid":
        w.Write([]byte("<html><head></head><body>invalid password</body><html>"))
        return
    }
    
    w.Write([]byte(`<html><head><meta name="uppertitle" content="login page"></head><body>welcome<br>enter your password here:<br><br><a href="http://`+cnf.Root+`/ds/v2-`+s.getregion()+`/sa/login.kbd">keyboard</a></body></html>`))
}

func sa_login_kbd(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := sessions[sid]
    in := r.Header.Get("X-Email-Addr")
    
    // sanity check
    if s.is_unregistered || s.is_logged_in {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    if v, err := verifyUserDsi(s.userid, in); err != nil {
        // handle error
        errorlog.Printf("while verifying user %d, %v", s.userid, err)
        w.Header()["X-DSi-Forwarder"] = []string{"http://" + cnf.Root + "/ds/v2-" + s.getregion() + "/sa/login.htm?ret=error"}
        return
    } else if !v {
        w.Header()["X-DSi-Forwarder"] = []string{"http://" + cnf.Root + "/ds/v2-" + s.getregion() + "/sa/login.htm?ret=invalid"}
        return
    }

    if err := updateUserLastLogin(s.userid, s.ip); err != nil {
        errorlog.Printf("while updating last login ip: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    s.is_logged_in = true
    sessions[sid] = s

    w.Header()["X-DSi-Forwarder"] = []string{"http://" + cnf.Root + "/ds/v2-" + s.getregion() + "/sa/success.htm"}
    w.WriteHeader(http.StatusOK)
}

// sa: successfully logged in/registered
func sa_success(w http.ResponseWriter, r *http.Request) {
    s := sessions[r.Header.Get("X-Dsi-Sid")]

    w.Write([]byte("<html><head></head><body>you have successfully logged in! (id " + strconv.Itoa(s.userid) + ")</body></html>"))
}

// sa: registration page
func sa_reg(w http.ResponseWriter, r *http.Request) {
    s := sessions[r.Header.Get("X-Dsi-Sid")]
    ret := r.URL.Query().Get("ret")
    if !s.is_unregistered { // user must be unregistered
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    switch ret {
    case "error":
        w.Write([]byte("<html><head></head><body>an error occurred</body></html>"))
        return
    }
    
    w.Write([]byte(`"<html><head><meta rel="stylesheet" href="http://` + cnf.Root + `/css/ds/basic.css"><meta name="uppertitle" content="registration"></head><body>welcome!<br>since you don't have a user account yet, you need to register<br><a href="http://` + cnf.Root + `/ds/v2-` + s.getregion() + `/sa/register.kbd">click here to register</a></body></html>`))
}

func sa_reg_kbd(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := sessions[sid]
    in := r.Header.Get("X-Email-Addr")
    if !s.is_unregistered { // user must be unregistered
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    id, err := registerUserDsi(in, s.fsid, s.ip)
    if err != nil {
        w.Header()["X-DSi-Forwarder"] = []string{ub(s.getregion(), "sa/register.htm?ret=error")}
        return
    }

    s.userid = id
    s.is_unregistered = false
    s.is_logged_in = true

    sessions[sid] = s

    w.Header()["X-DSi-Forwarder"] = []string{ub(s.getregion(), "sa/success.htm")}
    w.WriteHeader(http.StatusOK)
}