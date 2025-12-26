package main

import (
	"crypto/rsa"
	"crypto/x509"

	"io/fs"
	"os"

	"encoding/json"
	"html/template"

	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
    // do not change: der format rsa public key generated from
    // flipnote private key
    // this is a public key. it cannot be used to sign flipnotes, only verify the signature on flipnotes
    fnpub = []byte{0x30, 0x81, 0x9F, 0x30, 0x0D, 0x06, 0x09, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x01, 0x05, 0x00, 0x03, 0x81, 0x8D, 0x00, 0x30, 0x81, 0x89, 0x02, 0x81, 0x81, 0x00, 0xC2, 0x3C, 0xBC, 0x13, 0x2F, 0xAA, 0x12, 0x7E, 0x5B, 0xFE, 0x82, 0x3C, 0xB0, 0x8B, 0xFB, 0x0C, 0xD1, 0x35, 0x01, 0xF7, 0x4C, 0x6A, 0x3A, 0xFB, 0x82, 0xA6, 0x37, 0x6E, 0x11, 0x38, 0xCF, 0xA0, 0xDD, 0x85, 0xC0, 0xC7, 0x9B, 0xC4, 0xD8, 0xDD, 0x28, 0x8A, 0x87, 0x53, 0x20, 0xEE, 0xE0, 0x0B, 0xEB, 0x43, 0xA0, 0x43, 0x25, 0xCE, 0xA0, 0x29, 0x46, 0xD9, 0xD4, 0x4D, 0xBB, 0x04, 0x66, 0x68, 0x08, 0xF1, 0xF8, 0xF7, 0x34, 0x11, 0x6F, 0xEC, 0xC0, 0x33, 0xA3, 0x3D, 0x12, 0x31, 0xF0, 0x43, 0xA0, 0x40, 0x06, 0xBD, 0x2E, 0xD9, 0x37, 0x05, 0xEF, 0x11, 0xA0, 0xDA, 0xE4, 0x3D, 0x30, 0x15, 0xB3, 0xF4, 0x07, 0xDB, 0x55, 0x0F, 0x75, 0x36, 0x37, 0xEB, 0x35, 0x6A, 0x34, 0x7F, 0xB5, 0x0F, 0x99, 0xF7, 0xEF, 0xD5, 0x5B, 0xE2, 0xC6, 0x64, 0xE4, 0xD4, 0x10, 0xAD, 0x6A, 0xF6, 0x71, 0x07, 0x02, 0x03, 0x01, 0x00, 0x01}
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

func (e *env) initrsa() error {
    pub, err := x509.ParsePKIXPublicKey(fnpub)
    if err != nil {
        return err
    }
    
    ap, ok := pub.(*rsa.PublicKey)
    if !ok {
        return ErrNotRsaPubKey
    }
    
    e.fnkey = ap
    return nil
}


func initenv() (*env, error) {
    e := &env{
        sessions: make(map[string]*Session),
    }
    
    if err := e.load_config(false); err != nil {
        return nil, err
    }
    
    if err := e.load_html(false); err != nil {
        errorlog.Printf("failed to load html assets: %v", err)
    }
    if err := e.load_assets(false); err != nil {
        errorlog.Printf("failed to load other assets: %v", err)
    }
    if err := e.load_menus(false); err != nil {
        errorlog.Printf("failed to load menus: %v", err)
    }
    
    db, err := pgxpool.New(context.Background(), fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", e.cnf.DB.Host, e.cnf.DB.Port, e.cnf.DB.User, e.cnf.DB.Pass, e.cnf.DB.Name))
    if err != nil {
        return nil, err
    }
    
    if err := db.Ping(context.Background()); err != nil {
        return nil, err
    }
    
    e.pool = db
    infolog.Printf("connected to database (user=%s db=%s)", e.cnf.DB.User, e.cnf.DB.Name)
    
    if err := e.initrsa(); err != nil {
        errorlog.Printf("failed to initialize rsa key: %v", err)
    }
    
    go pruneSids(e.sessions)

    return e, nil
}