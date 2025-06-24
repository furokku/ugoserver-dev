package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
)

var (
    cnf = Configuration{}
    cf = "config.json"

	templates *template.Template
	menus = make(map[string]Ugomenu)
    texts = make(map[string]string)
)

// Load config file
func load_config(reload bool) error {
    if reload {
        infolog.Println("load_config: manual reload requested")
    }

    if len(os.Args) > 1 {
        cf = os.Args[1]
        if len(os.Args) > 2 {
            warnlog.Println("load_config: extra arguments were passed; only the first arg specifies config file, rest are ignored")
        }
    }

    cb, err := os.ReadFile(cf)
    if err != nil {
        return err
    }

    temp := Configuration{}
    err = json.Unmarshal(cb, &temp)
    if err != nil {
        return err
    }

    cnf = temp
    infolog.Printf("load_config: loaded %s", cf)
    return nil
}

// load html templates from static/template/*.html
//
// > parsing lots of files into one *template.Template produced weird results, so
// > they are stored in a map
// (no longer)
func load_templates(reload bool) error {
    if reload {
        infolog.Println("load_templates: manual reload requested")
    }
	
	// clear map contents
	//if reload {
	//	templates = make(map[string]*template.Template)
	//}
    // Better way to do this, as this will break things if it
    // fails to hot-reload
    
    //rd, err := os.ReadDir(cnf.Dir + "/static/template")
    //if err != nil {
    //    return err
    //}
    //for _, tpl := range rd {
    //    if tpl.IsDir() {
    //        continue
    //    }
    //    name := strings.Split(tpl.Name(), ".")[0]
    //    p, err := template.ParseFiles(fmt.Sprintf("%s/static/template/%s", cnf.Dir, tpl.Name()))
    //    if err != nil {
    //        errorlog.Printf("load_template: error parsing %s: %v\n", name, err)
    //        continue
    //    }
    //    temp[name] = p
    //}
    
    temp, err := template.ParseGlob(fmt.Sprintf("%v/static/template/*.html", cnf.Dir))
    if err != nil {
        return err
    }
    
    templates = temp
    infolog.Printf("load_template: loaded %d html templates%v", len(templates.Templates()), templates.DefinedTemplates())
    
    return nil
}

// load menus from static/menu/*.json
func load_menus(reload bool) error {
    if reload {
        infolog.Println("load_menus: manual reload requested")
    }
	
	//clear map contents before reloading
	//if reload {
	//	menus = make(map[string]Ugomenu)
	//}
    // Better way to do this, as this will break things if it
    // fails to hot-reload
    temp := make(map[string]Ugomenu)

    rd, err := os.ReadDir(cnf.Dir + "/static/menu")
    if err != nil {
        return err
    }
    for _, menu := range rd {
        if menu.IsDir() { // ignore subdirs
            continue
        }
        name := strings.Split(menu.Name(), ".")[0]
        bytes, err := os.ReadFile(fmt.Sprintf("%s/static/menu/%s", cnf.Dir, menu.Name()))
        if err != nil {
            errorlog.Printf("load_menu: error reading %s: %v", name, err)
            continue
        }
        tu := Ugomenu{}
        err = json.Unmarshal(bytes, &tu)
        if err != nil {
            errorlog.Printf("load_menu: error parsing %s: %v", name, err)
            continue
        }

        temp[name] = tu
    }

    menus = temp
    infolog.Printf("load_menu: loaded %d ugomenus", len(menus))
    
    return nil
}

// load text files from static/txt/*.txt
func load_text(reload bool) error { // finish
    if reload {
        infolog.Println("load_texts: manual reload requested")
    }
    
    temp := make(map[string]string)

    rd, err := os.ReadDir(cnf.Dir + "/static/text")
    if err != nil {
        return err
    }
    for _, text := range rd {
        name := strings.Split(text.Name(), ".")[0]

    }
    
    return nil
}