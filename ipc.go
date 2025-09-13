package main

import (
	"fmt"
	"strings"

	"io"
	"net"
	"time"
)

var ac = make(map[int]*net.Conn)

func newIpcListener(sf string, c cmdHandler) *ipcListener {
    s := &ipcListener{
        quit: make(chan interface{}),
    }
    l, err := net.Listen("unix", sf)
    if err != nil {
        errorlog.Fatalf("net.Listen: %v", err)
    }
    s.listener = l
    s.wg.Add(1)
    go s.serve(c)
    return s
}

func newCmdHandler() *cmdHandler {
    c := make(cmdHandler) // Not really a handler in the true sense
    
    return &c
}

func (c cmdHandler) register(n string, h cmdHandlerFunc) {
    c[n] = h
}

func (s *ipcListener) stop() {
    close(s.quit)
    s.listener.Close()
    s.wg.Wait()
}

func (s *ipcListener) serve(c cmdHandler) {
    id := 0
    defer s.wg.Done()

    for {
        conn, err := s.listener.Accept()
        if err != nil {
            select {
            case <-s.quit:
                for k, ic := range ac {
                    (*ic).Close()
                    delete(ac, k)
                }
                return
            default:
                errorlog.Printf("socket accept error: %v", err)
            }
        } else {
            id++
            ac[id] = &conn
            s.wg.Add(1)
            go func() {
                s.ipc(id, conn, c)
                s.wg.Done()
            }()
            infolog.Printf("ipc: new conn id %d", id)
        }
    }
}

func (s *ipcListener) ipc(id int, conn net.Conn, c cmdHandler) {
    defer conn.Close()

    buf := make([]byte, 4096)
    var resp string

    for {
        n, err := conn.Read(buf)
        if err != nil && err != io.EOF {
            errorlog.Printf("ipc: read error: %v", err)
            return
        }
        if n == 0 {
            return
        }
        req := string(buf[:n])

        args := strings.Split(req, " ")
        if len(args) == 0 {
            resp = "ok"
        } else {
            f, ok := c[args[0]]
            if !ok {
                resp = "unknown command " + args[0]
            } else {
                s := time.Now()
                resp = fmt.Sprintf("%s: %s (%dms)", args[0], f(args[1:]), time.Since(s).Milliseconds())
            }
        }
        infolog.Printf("ipc: %d ran %v", id, req)
        io.WriteString(conn, resp)
    }
}