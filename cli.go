package main

import (
	"fmt"
	"strconv"
	"time"
)

const (
	WHITELIST_USAGE string = "usage: whitelist add|del|query FSID"
	RELOAD_USAGE string = "usage: reload web|menus|assets"
    BAN_USAGE string =  "usage: ban [-f] FSID/IP duration message"
)

// cli handlers
//
// These should all follow the cmdHandlerFunc type (func([]string) string).

func (e *env) whitelist(args []string) string {
    
	if len(args) != 2 {
		return WHITELIST_USAGE
	}
	
	fsid := args[1]
	if !fsid_match.MatchString(fsid) {
		return fmt.Sprintf("%s is not an fsid", fsid)
	}
    
    switch args[0] {
    case "add":
        if err := whitelistAddFsid(e.pool, fsid); err != nil {
            errorlog.Printf("while adding to whitelist: %v", err)
            return err.Error()
		}
        
        return fmt.Sprintf("added %s", fsid)

    case "del":
        if err := whitelistDelFsid(e.pool, fsid); err != nil {
            errorlog.Printf("while removing from whitelist: %v", err)
            return err.Error()
        }
        
        return fmt.Sprintf("removed %s", fsid)
        
    case "query":
        if w, err := whitelistQueryFsid(e.pool, fsid); err != nil {
            errorlog.Printf("while querying whitelist: %v", err)
            return err.Error()
        } else {
            return fmt.Sprintf("%s %v", fsid, w)
        }
        
    default:
        return WHITELIST_USAGE
    }
}

// reload static content (ugomenus, html templates) and config
func (e *env) reload(args []string) string {
	
	if len(args) != 1 {
		return RELOAD_USAGE
	}
	
	switch args[0] {
	case "menus":
		if err := e.load_menus(true); err != nil {
            errorlog.Printf("load_menus: %v", err)
            return "internal error; check logs (load_menus)"
        }
		return "ok"

	case "web":
		if err := e.load_html(true); err != nil {
            errorlog.Printf("load_html: %v", err)
            return "internal error; check logs (load_html)"
        }
		return "ok"
        
	case "assets":
		if err := e.load_assets(true); err != nil {
            errorlog.Printf("load_assets: %v", err)
            return "internal error; check logs (load_assets)"
        }
		return "ok"
        
    default:
        return RELOAD_USAGE
	}

}

// ban a console either by IP or FSID;
// note: IP bans are solely that, per IP. if the public IP of a user changes,
// they will have unimpaired access to the service
// TODO -f to issue if already banned
func (e *env) ban(args []string) string {

    if len(args) != 3 {
        return BAN_USAGE
    }
    
    t := time.Now()
    
    target := args[0]
    if !fsid_match.MatchString(target) && !ip_match.MatchString(target) {
        return fmt.Sprintf("%s is not an ip or fsid", target)
    }

    d := dur_match.FindAllString(args[1], -1)
    if len(d) == 0 {
        return BAN_USAGE
    }
    msg := args[2]

    for _, iv := range d {
        n := iv[:len(iv)-1]
        c := iv[len(iv)-1]
        
        i, err := strconv.Atoi(n)
        if err != nil {
            errorlog.Printf("ban; while calling Atoi: %v", err)
            return "internal error; check logs (strconv.Atoi)"
        }
        
        switch c {
        case 'm':
            t = t.Add(time.Minute * time.Duration(i))
        case 'h':
            t = t.Add(time.Hour * time.Duration(i))
        case 'd':
            t = t.Add(time.Hour * time.Duration(i * 24))
        case 'w':
            t = t.Add(time.Hour * time.Duration(i * 24 * 7))
        }
    }
    
    if err := issueBan(e.pool, "console", t, target, msg, false); err != nil {
        errorlog.Printf("while banning %s: %v", target, err)
        return err.Error()
    }
    return fmt.Sprintf("banned %s until %s for %s", target, t.Format(time.DateTime), msg)
}

// pardon a user's ban either by ban id (specific)
// or IP/FSID (most recent active ban)
func pardon(args []string) string {
    return "wip"
}

// view/modify user-created content
func movie(args []string) string {
    return "wip"
}
func channel(args []string) string {
    return "wip"
}

// set/save configuration values while running
func config(args []string) string {
    return "wip"
}