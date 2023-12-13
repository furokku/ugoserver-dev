package main

import (
    "fmt"
    "time"
    "log"
    "encoding/base64"
    "strings"
    "golang.org/x/text/encoding/unicode"
    "math/rand"
    cryptoRand "crypto/rand"
)


// self explanatory, i think
func randBytes(count int) []byte {
    buf := make([]byte, count)
    cryptoRand.Read(buf)

    return buf
}


// self explanatory, again, i think
func randAsciiString(count int) string {
    chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

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
func encUTF16LE(data any) []byte {
    utf16 := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()

    var encoded []byte
    var err error

    switch data := data.(type) {
    case string:
        encoded, err = utf16.Bytes([]byte(data))
    case []byte:
        encoded, err = utf16.Bytes(data)
    }
    if err != nil {
        log.Printf("error encoding string to utf-16le %v", err)
        return []byte{}
    }

    return encoded
}


func decUTF16LE(data []byte) []byte {
    utf16 := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()

    decoded, err := utf16.Bytes(data)
    if err != nil {
        log.Printf("error decoding utf16le data %v", err)
        return []byte{}
    }

    return decoded
}


func decReqUsername(username string) string {
    utf16, err := base64.RawStdEncoding.DecodeString(username)
    if err != nil {
        log.Printf("decReqUsername(): failed to decode string %v with error %v", username, err)
        return ""
    }

    return string(decUTF16LE(utf16))
}


// issue a unique sid to the flipnote client
func genUniqueSession() string {
    var sid string

    for {
        sid = randAsciiString(32)
        if _, ok := sessions[sid]; !ok {
            return sid
        }
    }
}


// delete sessions issued 2h ago
func pruneSids() {
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

    if count > 999 {
        log.Printf("editCountPad(): error: edit count larger than 999 (%v), setting to 0", count)
        return "000"
    }

    return fmt.Sprintf("%03d", count)
}
