package main

import (
	"fmt"
)

const (
	WHITELIST_USAGE string = "usage: whitelist add|del|query FSID"
	RELOAD_USAGE string = "usage: reload menu|template"
    BAN_USAGE string =  "usage: ban FSID/IP duration message"
)

// cli handlers:
// runs when various commands in the console are invoked;
// for clarity, the function should be named the same as
// the command whenever possible

func whitelist(args []string) string {
    
	if len(args) != 2 {
		return WHITELIST_USAGE
	}
	
	fsid := args[1]
	if !fsid_match.MatchString(fsid) {
		return fmt.Sprintf("%s is not a valid fsid", fsid)
	}
    
    switch args[0] {
    case "add":
        if err := whitelistAddFsid(fsid); err != nil {
            errorlog.Printf("while adding to whitelist: %v", err)
            return err.Error()
		}
        
        return fmt.Sprintf("added %s", fsid)

    case "del":
        if err := whitelistDelFsid(fsid); err != nil {
            errorlog.Printf("while removing from whitelist: %v", err)
            return err.Error()
        }
        
        return fmt.Sprintf("removed %s", fsid)
        
    case "query":
        if w, err := whitelistQueryFsid(fsid); err != nil {
            errorlog.Printf("while querying whitelist: %v", err)
            return err.Error()
        } else {
            return fmt.Sprintf("%s %v", fsid, w)
        }
        
    default:
        return WHITELIST_USAGE
    }
}


func reload(args []string) string {
	
	if len(args) != 1 {
		return RELOAD_USAGE
	}
	
	switch args[0] {
	case "menu":
		load_menus(true)
		return ""

	case "template":
		load_templates(true)
		return ""
        
    default:
        return RELOAD_USAGE
	}

}

func ban(args []string) string {

}

func pardon(args []string) string {

}

func stat(args []string) string {
	
}