package main

import (
	"fmt"

	"bufio"
	"io"
	"net"

	"os"
	"os/signal"
)

func main() {
    
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt)
    

    sf := "/tmp/ugoserver.sock" // default socket path

    conn, err := net.Dial("unix", sf)
    if err != nil {
        panic(err)
    }

    go func(c net.Conn) {
        sig := <- sigs
        c.Close()
        
        fmt.Printf("caught %v, exiting\n", sig)
        os.Exit(0)
    }(conn)
    
    fmt.Printf("connected to server @ %s\n", sf)
    
    for {
        fmt.Print("> ")
        s := bufio.NewScanner(os.Stdin)
        s.Scan()
        if err := s.Err(); err != nil {
            panic(err)
        }

        io.WriteString(conn, s.Text())

        buf := make([]byte, 1048576) // read at most 1MiB, this should never be too little
        n, err := conn.Read(buf)
        if err != nil && err != io.EOF {
            panic(err)
        }

        fmt.Printf("%s\n", string(buf[:n]))
    }
}