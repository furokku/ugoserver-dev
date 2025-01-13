package main

import (
	"crypto/rand"
	"encoding/base64"

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


func randBytes(size int) []byte {
    buf := make([]byte, size)
    rand.Read(buf)

    return buf
}

func randAsciiString(size int) string {
    // cleaner
    buf := randBytes(size)
    for i, v := range buf {
        buf[i] = ASCII_CHARS[int(v) % len(ASCII_CHARS)]
    }
    
    return string(buf)
}

// quick base64 + utf16le encode
func q(s string) string {
    return base64.StdEncoding.EncodeToString(encUTF16LE(s))
}

func qd(s string) string {
    bytes := make([]byte, base64.StdEncoding.DecodedLen(len(s)))

    _, err := base64.StdEncoding.Decode(bytes, []byte(s))
    if err != nil {
        warnlog.Printf("failed to decode string %s with error %v", s, err)
        return ""
    }

    return string(decUTF16LE(bytes))
}

// nas response uses base64 with * and -
// in place of = and + due to url reserved chars
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

func editCountPad(count uint16) string {

    for count > 999 {
        warnlog.Printf("edit count larger than 999 (%v), looping back", count)
        count = count - 999
    }

    return fmt.Sprintf("%03d", count)
}

func reverse[T comparable](a []T) []T {
    r := make([]T, len(a))
    for i := len(a)/2-1; i >= 0; i-- {
        o := len(a)-1-i

        r[i], r[o] = a[o], a[i]
    }
    return r
}

func btoi(b bool) int {
    if b {
        return 1
    }
    return 0
}

func age(s string) int {
    t, err := time.Parse("20060102", s)
    if err != nil {
        return 0
    }

    return int(time.Since(t).Hours())/8760
}

func (f flipnote) TMB() (tmb, error) {
    buf := make([]byte, 0x6A0)
    path := fmt.Sprintf("%s/movies/%d.ppm", cnf.StoreDir, f.id)

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

// url builder
// takes in region and whatever you want after /ds/v2-xx/
// and spits out ready to use url
func ub(reg string, p string) string {
    return cnf.URL + "/ds/v2-" + reg + "/" + p
}