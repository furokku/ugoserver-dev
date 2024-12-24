package main

import (
    "net"
    "io"
)

func newIpcListener(sf string) *ipcListener {
    s := &ipcListener{
        quit: make(chan interface{}),
    }
    l, err := net.Listen("unix", sf)
    if err != nil {
        errorlog.Fatalf("net.Listen: %v", err)
    }
    s.listener = l
    s.wg.Add(1)
    go s.serve()
    return s
}

func (s *ipcListener) stop() {
    close(s.quit)
    s.listener.Close()
    s.wg.Wait()
}

func (s *ipcListener) serve() {
    defer s.wg.Done()

    for {
        conn, err := s.listener.Accept()
        if err != nil {
            select {
            case <-s.quit:
                return
            default:
                errorlog.Printf("socket accept error: %v", err)
            }
        } else {
            s.wg.Add(1)
            go func() {
                ipc(conn)
                s.wg.Done()
            }()
        }
    }
}

func ipc(conn net.Conn) {
    defer conn.Close()

    buf := make([]byte, 4096)

    for {
        n, err := conn.Read(buf)
        if err != nil && err != io.EOF {
            errorlog.Printf("read error %v", err)
            return
        }
        if n == 0 {
            return
        }
        req := string(buf[:n])
        resp := cmd(req)
        infolog.Printf("cmd: %v", req)
        io.WriteString(conn, resp)
    }
}
