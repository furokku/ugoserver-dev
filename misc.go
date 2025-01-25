package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"golang.org/x/text/encoding/unicode"

	"fmt"
	"os"
	"strings"

	"time"
)

var (
    utf16d = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
    utf16e = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
)

const (
    ASCII_CHARS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
)


// randBytes() returns n bytes, at random
func randBytes(n int) []byte {
    buf := make([]byte, n)
    rand.Read(buf)

    return buf
}

// randAsciiString() returns a random string with length n and characters from ASCII_CHARS
func randAsciiString(n int) string {
    // cleaner
    buf := randBytes(n)
    for i, v := range buf {
        buf[i] = ASCII_CHARS[int(v) % len(ASCII_CHARS)]
    }
    
    return string(buf)
}

// q() is a shortcut to encode a string to base64+utf16le
func q(s string) string {
    return base64.StdEncoding.EncodeToString(encUTF16LE(s))
}

// qd() is a quick decoder for base64+utf16le text
// Only used for usernames in sessions
func qd(s string) string {
    dec := make([]byte, base64.StdEncoding.DecodedLen(len(s)))

    _, err := base64.StdEncoding.Decode(dec, []byte(s))
    if err != nil {
        warnlog.Printf("failed to decode string %s with error %v", s, err)
        return ""
    }

    return string(stripnull(decUTF16LE(dec))) // Username is null padded, fix that
}

// stripnull() will remove any trailing null bytes from the input
func stripnull(in []byte) []byte {
    return bytes.Trim(in, "\x00")
}

// nasDecode() removes the url-safe versions of url-unsafe characters in the input
// and decodes it from base64
func nasDecode(data string) (string, error) {
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
        return "", err
    }

    return string(decoded), nil
}

// nasEncode() encodes a string to base64 and replaces url-unsafe characters in the output
// with url-safe ones
func nasEncode(data any) string {
    var encoded string

    switch data := data.(type) {
    case string:
        encoded = base64.StdEncoding.EncodeToString([]byte(data))
    case []byte:
        encoded = base64.StdEncoding.EncodeToString(data)
    default:
        return ""
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

// encUTF16LE() converts utf8 bytes to utf16le
// labels, messages in flipnote studio require to be represented in utf16le
func encUTF16LE(data any) []byte {
    var encoded []byte
    var err error

    switch data := data.(type) {
    case string:
        encoded, err = utf16e.Bytes([]byte(data))
    case []byte:
        encoded, err = utf16e.Bytes(data)
    default:
        return nil
    }
    if err != nil {
        warnlog.Printf("error encoding string to utf-16le %v", err)
        return nil
    }

    return encoded
}

// decUTF16LE() converts utf16le bytes to utf8 bytes
// make sure to strip the output of null bytes if necessary, as some things are padded like that
func decUTF16LE(data []byte) []byte {
    decoded, err := utf16d.Bytes(data)
    if err != nil {
        warnlog.Printf("error decoding utf16le data %v", err)
        return nil
    }

    return decoded
}

// countPages() takes a total, t, and a number of elements per page, e,
// returning the max amount of pages for e elements per page with t total elements
func countPages(t int, e int) int {
    pages := t / e
    if t % e > 0 {
        pages += 1
    }

    return pages
}

// editCountPad() takes an integer and pads it to 3 digits if necessary
// example: 001, 067, 900.
func editCountPad(count uint16) string {

    for count > 999 {
        // Edge case, in theory should not happen
        warnlog.Printf("edit count larger than 999 (%v), looping back", count)
        count = count - 999
    }

    return fmt.Sprintf("%03d", count)
}

// reverse() will return an array with all elements in reverse order
func reverse[T comparable](a []T) []T {
    r := make([]T, len(a))
    for i := len(a)/2-1; i >= 0; i-- {
        o := len(a)-1-i

        r[i], r[o] = a[o], a[i]
    }
    return r
}

// btoi()
func btoi(b bool) int {
    if b {
        return 1
    }
    return 0
}

// age() calculates the age in years of a user from a session
func (s Session) age() int {
    t, err := time.Parse("20060102", s.Birthday)
    if err != nil {
        return 0
    }

    return int(time.Since(t).Hours())/8760
}

// tmb() returns the first 0x6a0 bytes of a ppm in order to embed it in a menu
func (f Movie) tmb() ([]byte, error) {
    buf := make([]byte, 0x6A0)
    path := fmt.Sprintf("%s/movies/%d.ppm", cnf.StoreDir, f.ID)

    file, err := os.Open(path)
    if err != nil {
        errorlog.Printf("failed to open %v: %v", path, err)
        return nil, err
    }
    _, err = file.Read(buf)
    if err != nil {
        errorlog.Printf("failed to read %v: %v", path, err)
        return nil, err
    }

    return buf, nil
}

// ub() (url builder) is a method on session to automatically insert the correct region
// and make a url from argument
func (s Session) ub(p string) string {
    return fmt.Sprintf("http://%s/ds/v2-%s/%s", cnf.Root, s.getregion(), p)
}

// returncode() returns an http handler which only writes one header to all responses
func returncode(code int) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(code)
    }

    return fn
}