package main

// ugotool stress:
// Pseudo-benchmark for ugoserver (in theory for any hatena server)
//
// the 'how'
// "workers" - simulated clients
// each one will have their own made up sid/fsid/etc
// worker fsids should start with a specific value so they can easily
// be identified (i.e. 5000BEEFXXXXXXXX)
// TODO: make this get first X rows from database and keep that in memory
// to have an index of what to fetch using the fake DSis
//
// allow workers to use a pool of ips?
// unix socket for response verification w/sha256?
// config options: allow signup, allow post, allow comment,
// enable response auth, periodically rotate workers or increase their amount?
// whether to go through NAS
// delay

import (
	"crypto/rand"

	"encoding/hex"

	"fmt"
	"regexp"
	"strings"
	"time"

	"net/http"
)

const (
	// stress actions
	GET_INDEX = iota
	GET_CHANNELS
	GET_FEED_NEW
	GET_FEED_TOP
	GET_FEED_HOT
	GET_CH_FEED_NEW
	GET_CH_FEED_TOP
	GET_CH_FEED_HOT
	JUMP // throw in random jump code
	GET_MOVIE
	GET_REPLY
	GET_MOVIE_DETAIL
	GET_MOVIE_DETAIL_REPLIES
	REAUTH // do /ds/v2-us/auth over again and new sid
	POST_MOVIE // posts to random channel
	POST_REPLY // posts reply to random movie on feed/new
)

func worker(id int) {
	fmt.Printf("worker %d: started\n", id)
	mac := newmac()
	fsid := "5000BEEF" + mac[6:]

	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	// initialize
	resp1, err := cl.Get("http://127.0.0.1:9000/ds/v2-us/auth")
	if err != nil || resp1.Status != "200 OK" {
		fmt.Printf("worker %d error (init1): %v", id, err)
		return
	}
	
	sid := resp1.Header.Get("X-Dsi-Sid")
	
	auth2r, err := http.NewRequest("POST", "http://127.0.0.1:9000/ds/v2-us/auth", nil)
	if err != nil {
		fmt.Printf("worker %d error (init2): %v", id, err)
		return
	}
	
	auth2r.Header["X-DSi-SID"] = []string{sid}
	auth2r.Header["X-DSi-MAC"] = []string{mac}
	auth2r.Header["X-DSi-ID"]  = []string{fsid}
	auth2r.Header["X-DSi-User-Name"] = []string{"dwBvAHIAawBlAHIAcgBlAGEAbAA="} // `workerreal`
	auth2r.Header["X-DSi-Region"] = []string{"1"}
	auth2r.Header["X-DSi-Lang"] = []string{"en"}
	auth2r.Header["X-DSi-Country"] = []string{"US"}
	auth2r.Header["X-DSi-Color"] = []string{"039e"}
	auth2r.Header["X-DSi-DateTime"] = []string{time.Now().Format("2006-01-02_15:04:05")}
	auth2r.Header["X-DSi-Auth-Response"] = []string{"80085"}
	auth2r.Header["X-Ugomemo-Version"] = []string{"2"}
	auth2r.Header["X-Birthday"] = []string{"20010911"}
	
	resp2, err := cl.Do(auth2r)
	if err != nil || resp2.Status != "200 OK" {
		fmt.Printf("worker %d error (init2): %v", id, err)
		return
	}
	

	// register
	// Shouldn't matter to log in instead, all worker resources should be
	// deleted afterwards
	
	// main loop
	// go through random resources here
	for {

	}
}

// return random mac addr
func newmac() string {
	buf := make([]byte, 3)
	rand.Read(buf)
	
	return "006700" + strings.ToUpper(hex.EncodeToString(buf))
}

// returns a list of all links from an ugomenu or an htm with a target file extension
// (ppm, uls, npf, htm, or whatever)
func extractlink(menu []byte, target string) []string {
	re, _ := regexp.Compile(fmt.Sprintf("http://.+\\.%s[a-zA-Z0-9_?&=-]*", target))
	return re.FindAllString(string(menu), -1)
}

func signrsa(in []byte) []byte {
	//todo

	return nil
}