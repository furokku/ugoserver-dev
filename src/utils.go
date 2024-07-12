package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/encoding/unicode"
)

var (
    utf16d = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
    utf16e = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
    chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
)

func randBytes(size int) []byte {
    buf := make([]byte, size)
    rand.Read(buf)

    return buf
}


func randAsciiString(size int) string {
    // cleaner
    buf := randBytes(size)
    for i, v := range buf {
        buf[i] = chars[int(v) % len(chars)]
    }
    
    return string(buf)
}


// nas response uses base64 with * and -
// in place of = and + due to url reserved chars
func nasDecode(data string) string {
    decoded, err := base64.StdEncoding.DecodeString(strings.Map(func(r rune) rune {
        switch r {
        case '*':
            return '='
        case '-':
            return '+'
        case '.':
            return '/'
        default:
            return r
        }
    }, data))
    if err != nil {
        warnlog.Printf("(nas) decoding base64 string %v failed with error %v", data, err)
        return ""
    }

    return string(decoded)
}


func nasEncode(data any) string {
    var encoded string

    switch data := data.(type) {
    case string:
        encoded = base64.StdEncoding.EncodeToString([]byte(data))
    case []byte:
        encoded = base64.StdEncoding.EncodeToString(data)
    }

    return strings.Map(func(r rune) rune {
        switch r {
        case '=':
            return '*'
        case '+':
            return '-'
        case '/':
            return '.'
        default:
            return r
        }
    }, encoded)
}


// any text that is displayed on the screen in flipnote studio
// must be in UTF16-LE
func encUTF16LE(data any) []byte {
    var encoded []byte
    var err error

    switch data := data.(type) {
    case string:
        encoded, err = utf16e.Bytes([]byte(data))
    case []byte:
        encoded, err = utf16e.Bytes(data)
    }
    if err != nil {
        warnlog.Printf("error encoding string to utf-16le %v", err)
        return nil
    }

    return encoded
}


func decUTF16LE(data []byte) []byte {
    decoded, err := utf16d.Bytes(data)
    if err != nil {
        warnlog.Printf("error decoding utf16le data %v", err)
        return nil
    }

    return decoded
}


func decReqUsername(username string) string {
    bytes := make([]byte, base64.StdEncoding.DecodedLen(len(username)))

    _, err := base64.StdEncoding.Decode(bytes, []byte(username))
    if err != nil {
        warnlog.Printf("failed to decode string %v with error %v", username, err)
        return ""
    }

    decoded, err := utf16d.Bytes(bytes)
    if err != nil {
        warnlog.Printf("failed to decode utf16 from %v: %v", username, err)
        return ""
    }

    return string(decoded)
}


// issue a unique sid to the client
func genUniqueSession() string {
    var sid string

    for {
        sid = randAsciiString(32)
        if _, ok := sessions[sid]; !ok {
            return sid
        }
    }
}


// find expired sessions and delete them every
// 5 minutes
func pruneSids() {
    for {
        time.Sleep(5 * time.Minute)

        for k, v := range sessions {
            t := v.issued
            elapsed := time.Now().Unix() - t.Unix()
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
    pages := t / 50
    if t % 50 > 0 {
        pages += 1
    }

    return pages
}


// find offset for sql query based on current page
func findOffset(p int) int {
    return (p - 1) * 50
}


func editCountPad(count uint16) string {

    for count > 999 {
        warnlog.Printf("edit count larger than 999 (%v), looping back", count)
        count = count - 999
    }

    return fmt.Sprintf("%03d", count)
}
