package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"time"

	"slices"
	"strconv"
	"strings"
	// debug
	//"bytes"
)

const (
    // nas isn't used here so this can be garbage
    NAS_TOKEN string = "NDSflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocflocfloc"
    MELONDS_BSSID string = "00f077777777" // default melonAP bssid, catches emulators

    MSG_NO_SUPPORT string = "you are using an outdated version of flipnote studio.\nplease connect using the latest version."
    MSG_EARLY_ERROR string = "an error occurred during\nearly authentication."
    MSG_ERROR_REF string = "an error occured. try again later\nreference: "
)

var GAMECODES = []string{"KGUE", "KGUV", "KGUJ", "NTRJ"} // add tv-jp

// hatenaAuth handler authenticates clients after NAS on /ds/v2-xx/auth
func (e *env) hatenaAuth(w http.ResponseWriter, r *http.Request) {

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
        w.Header()["X-DSi-SID"] = []string{generateUniqueSession(e.sessions)}

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
            RegionCode:   region, // unset on rev2
            Region: getregion(region),
            Lang:     r.Header.Get("X-Dsi-Lang"), // unset on rev2
            Country:  r.Header.Get("X-Dsi-Country"), // unset on rev2
            Birthday: r.Header.Get("X-Birthday"),
            DateTime: r.Header.Get("X-Dsi-Datetime"), // unset on rev2
            Color:    r.Header.Get("X-Dsi-Color"), // unset on rev2
            
            IP: ip,
            Issued: time.Now(),
        }

        ref := sid[:6] + "_" + req.MAC[6:]

        if r, err := e.validate(req); err != nil {
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
        u, err := getUserByFsid(e.pool, req.FSID)
        if err != nil {
            switch err {
            case ErrNoUser:
                // ignore
            default:
                errorlog.Printf("while getting user: %v", err)
                w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
                w.Write(encUTF16LE(MSG_ERROR_REF + ref))
                return
            }
        }

        if u.ID == 0 {
            req.IsUnregistered = true
        } else {
            req.UserID = u.ID
            if ip == u.LastLoginIP {
                req.IsLoggedIn = true
            }
        }
        
        e.sessions[sid] = &req
        //debuglog.Printf("new session %v : %v\n", sid, sessions[sid])

        // if nothing else failed, set SID header
        // possible to add X-DSi-New/Unread-Notices here
        // for flashing NEW on inbox button
        w.Header()["X-DSi-SID"] = []string{sid}
        
        // todo: mail
    }

    w.WriteHeader(http.StatusOK)
}

// nosupport handler accept rev1/2 clients on /ds/[v2/]auth and rejects them with a message
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
func (e *env) nasAuth(w http.ResponseWriter, r *http.Request) {

    ip := r.Header.Get("X-Real-Ip")

    // check game code in http header
    gcdh := r.Header.Get("HTTP_X_GAMECD")
    if !slices.Contains(GAMECODES, gcdh) {
        warnlog.Printf("unknown gamecode %s", gcdh)
    w.WriteHeader(http.StatusBadRequest)
    return
    }

    body, err := io.ReadAll(r.Body)
    if err != nil {
        errorlog.Printf("failed to parse form from %v: %v", ip, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
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

            // check game code
	    //gcdr := nasRequest.Get("gamecd") // KGUE (U), KGUV (E), KGUJ (J)
            bssid := nasRequest.Get("bssid")
            // Emulator check
            // Most users who try to use an emulator won't
            // change the default AP BSSID, which will give awau
            // the fact that they're using one
            // Worth noting: this is bypassable if using own dns server
            // with custom NAS

            if bssid == MELONDS_BSSID { // 00:F0:77 mac is unassigned. 100% emulator
                if err := issueBan(e.pool, "auto", time.Now().Add(60 * time.Minute), ip, "emulator usage", true); err == ErrAlreadyBanned {
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
                    resp.Set("userid", nasEncode(randAsciiString(20)))

                default:
                    debuglog.Printf("nas action %s", action)
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
func (e *env) validate(a Session) (*Ban, error) {

    if ok, err := whitelistQueryFsid(e.pool, a.FSID); err != nil {
        return nil, err
    } else if ok {
        return nil, nil
    }

    if r, err := queryBan(e.pool, a.FSID); err != nil {
        return nil, err
    } else if r != nil {
        return r, ErrFsidBan
    }
    if r, err := queryBan(e.pool, a.IP); err != nil {
        return nil, err
    } else if r != nil {
        return r, ErrIpBan
    }

    if a.MAC[5:] != a.FSID[9:] {
        return nil, ErrAuthMacFsidMismatch
    }
    if a.FSID[9:] == "BF112233" {
        return nil, ErrAuthEmulatorId
    }
    if a.MAC == "0009BF112233" {
        return nil, ErrAuthEmulatorMac
    }
    if a.age() < 13 {
        return nil, ErrAuthUnderage
    }

    return nil, nil
}

// isSidValid() checks if a sid is taken
func isSidValid(sessions map[string]*Session, sid string) bool {
    if _, ok := sessions[sid]; ok {
        return true
    }
    return false
}

// genSid() returns a new, unusued session identifier
func generateUniqueSession(sessions map[string]*Session) string {
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
func pruneSids(sessions map[string]*Session) {
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
func getregion(r int) string {
    switch r {
    case 0:
        return "jp"
    case 1:
        return "us"
    case 2:
        return "eu"
    default:
        return "jp"
    }
}

// dsi_am middleware checks whether a user is logged in, and optionally redirects them
// to a log in page if they are not;
// note that the redirect functionality only works for html
func (e *env) dsi_am(next http.HandlerFunc, check_id bool, redirect bool) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        sid := r.Header.Get("X-Dsi-Sid")
        s := e.sessions[sid]
        // Not authenticated thru flipnote
        if !isSidValid(e.sessions, sid) {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        
        if check_id && !s.IsLoggedIn {
            if redirect {
                cutpath, _ := strings.CutPrefix(r.URL.Path, fmt.Sprintf("/ds/v2-%s/", s.Region))
                uq := r.URL.Query()
                
                rurl := fmt.Sprintf("%s?%s", cutpath, uq.Encode())

                //debuglog.Println(rurl)
                http.HandlerFunc(e.sa(rurl)).ServeHTTP(w, r)
                return
            }
            w.WriteHeader(http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    }
    
    return fn
}

// sa (secondary auth) returns a handler with optional redirect
// only index 0 from rd will be used
func (e *env) sa(rd... string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ret := r.URL.Query().Get("ret")
        
        //buf := new(bytes.Buffer)
        
        d, err := e.fillpage(r.Header.Get("X-Dsi-Sid"))
        if err != nil {
            errorlog.Printf("while filling page (sa): %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        // it was because i'm dumb as rocks
        //debuglog.Println(d) // is this nil?
        d["return"] = ret

        // takes priority
        if len(rd)>0 {

            // workaround: flipnote studio has a bug where it will parse abc.kbd?foo=bar.htm and read the .htm in the end
            // and will try to GET it instead of following the .kbd
            // do something about that here (url encode the redirect path and replace . with -)
            d["redirect"] = strings.ReplaceAll(url.QueryEscape(rd[0]), ".", "-")

            // also check rd query if login failed to not break the redirect chain
        } else if rdp := r.URL.Query().Get("rd"); rdp != "" {
            d["redirect"] = rdp
        } 

        if err := e.html.ExecuteTemplate(w/*buf*/, "auth.html", d); err != nil {
            errorlog.Printf("while executing template: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
        }
        
        //debuglog.Println(buf.String())
        //w.Write(buf.Bytes())
        w.WriteHeader(http.StatusOK)
    }
}

// sa_login_kbd handler takes the input from a keyboard POST and checks if the password is correct for the user
func (e *env) sa_login_kbd(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := e.sessions[sid]

    pw := r.Header.Get("X-Email-Addr")
    
    u := url.Values{}
    
    //debuglog.Println("got to login_kbd")
    
    // sanity check
    if s.IsUnregistered || s.IsLoggedIn {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    re := false
    rd, err := url.QueryUnescape(r.URL.Query().Get("rd"))
    if err == nil && rd != "" {
        re = true
        u.Add("rd", rd)
    }
    
    if v, err := verifyUserById(e.pool, s.UserID, pw, s.IP); err != nil {
        // handle error
        errorlog.Printf("while verifying user %d, %v", s.UserID, err)
        u.Add("ret", "error")
    } else if !v {
        u.Add("ret", "invalid")
    } else {
        s.IsLoggedIn = true
        u.Add("ret", "success")
    }

    if re && s.IsLoggedIn {
        w.Header()["X-DSi-Forwarder"] = []string{ub(e.cnf.Root, s.Region, strings.ReplaceAll(rd, "-", "."))}
    } else {
        w.Header()["X-DSi-Forwarder"] = []string{fmt.Sprintf("%s?%s", ub(e.cnf.Root, s.Region, "sa/auth.htm"), u.Encode())}
    }
    w.WriteHeader(http.StatusOK)
}

// sa_reg_kbd handler takes the input from a keyboard POST and registers the user
func (e *env) sa_reg_kbd(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := e.sessions[sid]

    pw := r.Header.Get("X-Email-Addr")
    
    u := url.Values{}

    if !s.IsUnregistered { // user must be unregistered
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    re := false
    rd, err := url.QueryUnescape(r.URL.Query().Get("rd"))
    if err == nil && rd != "" {
        re = true
        u.Add("rd", rd)
    }
    
    id, err := registerUserDsi(e.pool, s.Username, pw, s.FSID, s.IP)
    if err != nil {
        errorlog.Printf("while registering user: %v", err)
        u.Add("ret", "error")
    } else {
        s.UserID = id
        s.IsUnregistered = false
        s.IsLoggedIn = true

        u.Add("ret", "success")
    }

    if re {
        w.Header()["X-DSi-Forwarder"] = []string{ub(e.cnf.Root, s.Region, strings.ReplaceAll(rd, "-", "."))}
    } else {
        w.Header()["X-DSi-Forwarder"] = []string{fmt.Sprintf("%s?%s", ub(e.cnf.Root, s.Region, "sa/auth.htm"), u.Encode())}
    }
    w.WriteHeader(http.StatusOK)
}

func (e *env) api_auth(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/api/auth/login": // todo: fsid login
        body, err := io.ReadAll(r.Body)
        if err != nil {
            errorlog.Printf("while reading form body from auth/login: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        //debuglog.Println(string(body))

        lf, err := url.ParseQuery(string(body))
        if err != nil {
            errorlog.Printf("while parsing form body from auth/login: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            w.Write([]byte("sorry, an unexpected error occurred"))
            return
        }

        // todo: determine if first field input is a user id or fsid
        // for now assume id
        
        ids := lf.Get("userid")
        id, err := strconv.Atoi(ids)
        if err != nil {
            errorlog.Printf("while converting id string to int: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            w.Write([]byte("sorry, an unexpected error occurred"))
            return
        }
        
        // for now last login is only tracked on the dsi
        success, err := verifyUserById(e.pool, id, lf.Get("pw"))
        if err != nil {
            errorlog.Printf("while verifying user id: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            w.Write([]byte("sorry, an unexpected error occurred"))
            return
        }
        
        if success {
            // login cookie
            secret, err := newApiToken(e.pool, id)
            if err != nil {
                errorlog.Printf("while creating new api token for user %d: %v", id, err)
                w.WriteHeader(http.StatusInternalServerError)
                w.Write([]byte("sorry, an unexpected error occurred"))
                return
            }
            
            http.SetCookie(w, &http.Cookie{
                Name: "token",
                Value: secret,
                
                Domain: r.Host,
                Path: "/",

                MaxAge: 0,
                SameSite: http.SameSiteStrictMode,
            })

            w.Header().Add("Location", fmt.Sprintf("http://%s/ui/account.html", r.Host))
            w.WriteHeader(http.StatusSeeOther)
            return
        }
        
        // assume failure unless above returns
        w.Header().Add("Location", fmt.Sprintf("http://%s/ui/account.html?ret=invalidlogin", r.Host))
        w.WriteHeader(http.StatusSeeOther)
    }
}