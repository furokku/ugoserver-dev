package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {

    sf := "/tmp/ugoserver.sock" // default socket path

    cmd := os.Args[1]
    for _, arg := range os.Args[2:] {
        cmd += " "
        cmd += arg
    }

    fmt.Printf("< %s\n", cmd)

    conn, err := net.Dial("unix", sf)
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    io.WriteString(conn, cmd)

    buf := make([]byte, 1048576)
    n, err := conn.Read(buf)
    if err != nil && err != io.EOF {
        panic(err)
    }

    fmt.Printf("> %s\n", string(buf[:n]))
}
