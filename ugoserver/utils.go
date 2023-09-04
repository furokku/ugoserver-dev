package main

import (
    "time"
    "log"
    "encoding/base64"
    "strings"
    "golang.org/x/text/encoding/unicode"
    "math/rand"
    cryptoRand "crypto/rand"
)


func randBytes(count int) []byte {
    buf := make([]byte, count)
    cryptoRand.Read(buf)

    return buf
}


func randAsciiString(count int) string {
    buf := make([]rune, count)
    for i := range buf {
        buf[i] = chars[rand.Intn(len(chars))]
    }
    
    return string(buf)
}


// nas response uses base64 with * and -
// in place of = and + due to url reserved chars
func decode(str string) string {
    decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.ReplaceAll(str, "-", "+"), "*", "="))
    if err != nil {
        log.Printf("error decoding base64 string %v with error %v", str, err)
        return ""
    }

    return string(decoded)
}


func encode(data any) string {
    var encoded string

    switch data := data.(type) {
    case string:
        encoded = base64.StdEncoding.EncodeToString([]byte(data))
    case []byte:
        encoded = base64.StdEncoding.EncodeToString(data)
    }

    return strings.ReplaceAll(strings.ReplaceAll(encoded, "+", "-"), "=", "*")
}


// For error messages and labels
func encUTF16LE(str string) []byte {
    utf16 := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()

    encoded, err := utf16.Bytes([]byte(str))
    if err != nil {
        log.Printf("error encoding string to utf-16le %v with error %v", str, err)
        return []byte{}
    }

    return encoded
}


// issue a unique sid to the flipnote client
func genUniqueSession() string {
    var sid string

    for {
        sid = randAsciiString(32)
        if _, ok := sessions[sid]; !ok {
            break
        }
    }
    return sid
}


func pruneSessions() {
    for {
        time.Sleep(5 * time.Minute)

        for k, v := range sessions {
            t := v.issued
            elapsed := time.Now().Unix() - t
            if elapsed >= 7200 {
                delete(sessions, k)
            }
        }
    }
}


// find amount of pages possible
// based on total amount of flipnotes
// in the result
func countPages(t int) int {
    pages := t / 54
    if n := pages % 54; n > 0 {
        pages += 1
    }

    return pages
}


// find offset for sql query based on current page
func findOffset(p int) int {
    return (p - 1) * 54
}


// 4 byte padding for ugomenus
func padBytes(d []byte) []byte {
    var padded []byte = d

    if x := len(d) % 4; x != 0 {
        for i := 0; i < (4-x); i++ {
            padded = append(padded, nul)
        }

        return padded
    }

    return d
}
