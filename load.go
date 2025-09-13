package main

import (
	"io/fs"
	"os"

	"encoding/json"
	"html/template"

	"fmt"
	"strings"
)

// Load config file
func (e *env) load_config(reload bool) error {
    cf := "config.json"

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

    c := Configuration{}
    err = json.Unmarshal(cb, &c)
    if err != nil {
        return err
    }

    e.cnf = &c
    infolog.Printf("load_config: loaded %s", cf)
    
    return nil
}

// load html templates from static/template/*.html and css files
// in web_assets/
//
// > parsing lots of files into one *template.Template produced weird results, so
// > they are stored in a map
// (no longer)
func (e *env) load_html(reload bool) error {
    if reload {
        infolog.Println("load_html: manual reload requested")
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
    
    // html part
    h, err := template.ParseGlob(fmt.Sprintf("%v/assets/special/html/*.html", e.cnf.Dir))
    if err != nil {
        return err
    }
    
    e.html = h
    infolog.Printf("load_html: loaded %d html templates%v", len(e.html.Templates()), e.html.DefinedTemplates())

    return nil
}

func (e *env) load_assets(reload bool) error {
    if reload {
        infolog.Println("load_assets: manual reload requested")
    }

    a := make(map[string][]byte)
    fs.WalkDir(os.DirFS(e.cnf.Dir), "assets", walktomap(&a))

    e.assets = a
    infolog.Printf("load_assets: loaded %d other assets", len(e.assets))
    //fmt.Println(cache_assets)
    return nil
}

// load menus from assets/special/menus/*.json
func (e *env) load_menus(reload bool) error {
    if reload {
        infolog.Println("load_menus: manual reload requested")
    }
	
	//clear map contents before reloading
	//if reload {
	//	menus = make(map[string]Ugomenu)
	//}
    // Better way to do this, as this will break things if it
    // fails to hot-reload
    //m := make(map[string]Ugomenu)

    //rd, err := os.ReadDir()
    //if err != nil {
    //    return err
    //}
    //for _, menu := range rd {
    //    if menu.IsDir() { // ignore subdirs
    //        continue
    //    }
    //    name := strings.Split(menu.Name(), ".")[0]
    //    bytes, err := os.ReadFile(fmt.Sprintf("%s/static/menu/%s", cnf.Dir, menu.Name()))
    //    if err != nil {
    //        errorlog.Printf("load_menu: error reading %s: %v", name, err)
    //        continue
    //    }
    //    tu := Ugomenu{}
    //    err = json.Unmarshal(bytes, &tu)
    //    if err != nil {
    //        errorlog.Printf("load_menu: error parsing %s: %v", name, err)
    //        continue
    //    }

    //    m[name] = tu
    //}
    
    m := make(map[string]Ugomenu)
    fs.WalkDir(os.DirFS(e.cnf.Dir), "assets/special/menu", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            errorlog.Printf("load_menu: WalkDir passed an error to anon WalkFunc: %v", err)
            return err
        }
        if d.Name() == "ignore" && d.IsDir() {
            return fs.SkipDir
        }
        if d.IsDir() {
            return nil
        }
        
        name := strings.Split(d.Name(), ".")[0]
        jb, err := os.ReadFile(path)
        if err != nil {
            errorlog.Printf("load_menu (anonymous WalkFunc): failed to read %s: %v; skipping...", path, err)
            return nil
        }
        u := Ugomenu{}
        if err := json.Unmarshal(jb, &u); err != nil {
            errorlog.Printf("load_menu (anonymous WalkFunc): failed to parse %s: %v; skipping...", path, err)
            return err
        }
        
        m[name] = u

        return nil
    })

    e.menus = m
    infolog.Printf("load_menu: loaded %d ugomenus", len(e.menus))
    
    return nil
}


// Directory tree walking
// intended for this to be used with various different maps
func walktomap(m *map[string][]byte) (fs.WalkDirFunc) {
    return func(path string, d fs.DirEntry, err error)(error){
        if err != nil {
            errorlog.Printf("WalkDir passed an error to walktomap: %v", err)
            return err // do not continue if an error was already encountered
        }
        if d.Name() == "special" && d.IsDir() {
            return fs.SkipDir // skip /special and all of its contents
        }
        if d.IsDir() {
            return nil // ignore directories
        }
        
        shortpath, _ := strings.CutPrefix(path, "assets/")
        fc, err := os.ReadFile(path)
        if err != nil {
            errorlog.Printf("walktomap: failed to read %s: %v; skipping...", path, err)
            return nil
        }
        
        (*m)[shortpath] = fc // key short path (i.e. images/cat.npf) to file contents

        return nil
    }
}