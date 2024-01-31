package main

import (
    "log"
    "os"
)

var (
    colorReset = "\033[0m"

    debuglog = log.New(os.Stdout, "[debug] ", log.Lshortfile|log.Ldate|log.Ltime)
    infolog = log.New(os.Stdout, "[info] ", log.Lshortfile|log.Ldate|log.Ltime)
    warnlog = log.New(os.Stdout, "\033[33m[warn] " + colorReset, log.Lshortfile|log.Ldate|log.Ltime)
    errorlog = log.New(os.Stdout, "\033[31m[error] " + colorReset, log.Lshortfile|log.Ldate|log.Ltime)
)
