package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
)

var (
	templates = make(map[string]*template.Template)
	menus = make(map[string]Ugomenu)
)

// load html templates from static/template/*.html
//
// parsing lots of files into one *template.Template produced weird results, so
// they are stored in a map
func load_templates(reload bool) {
	
	// clear map contents
	if reload {
		templates = make(map[string]*template.Template)
	}

    rd, err := os.ReadDir(cnf.Dir + "/static/template")
    if err != nil {
        errorlog.Fatalf("load_template: %v\n", err)
    }
    for _, tpl := range rd {
        if tpl.IsDir() {
            continue
        }
        name := strings.Split(tpl.Name(), ".")[0]
        p, err := template.ParseFiles(fmt.Sprintf("%s/static/template/%s", cnf.Dir, tpl.Name()))
        if err != nil {
            errorlog.Printf("load_template: error parsing %s: %v\n", name, err)
            continue
        }
        templates[name] = p
    }
    infolog.Printf("load_template: loaded %d html templates\n", len(templates))
}

// load menus from static/menu/*.json
func load_menus(reload bool) {
	
	// clear map contents before reloading
	if reload {
		menus = make(map[string]Ugomenu)
	}

    rd, err := os.ReadDir(cnf.Dir + "/static/menu")
    if err != nil {
        errorlog.Fatalf("load_menu: %v\n", err)
    }
    for _, menu := range rd {
        if menu.IsDir() { // ignore subdirs
            continue
        }
        name := strings.Split(menu.Name(), ".")[0]
        bytes, err := os.ReadFile(fmt.Sprintf("%s/static/menu/%s", cnf.Dir, menu.Name()))
        if err != nil {
            errorlog.Printf("load_menu: error reading %s: %v\n", name, err)
            continue
        }
        tu := Ugomenu{}
        err = json.Unmarshal(bytes, &tu)
        if err != nil {
            errorlog.Printf("load_menu: error parsing %s: %v\n", name, err)
            continue
        }

        menus[name] = tu
    }
    infolog.Printf("load_menu: loaded %d ugomenus\n", len(menus))
}