package main

import (
	"net/http"
	"log"
	"fmt"
	"time"
	"math/rand"
	"strconv"
)

func Check(w http.ResponseWriter, r *http.Request, inter int) {
	// Grab string of interval
	interval := strconv.Itoa(inter)
	// Add required headers
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")
	w.Header().Add("X-Wii-Mail-Download-Span", interval)
	w.Header().Add("X-Wii-Mail-Check-Span", interval)

	// HMAC key most likely used for `chlng`
	// TODO: insert hmac thing
	// "ce4cf29a3d6be1c2619172b5cb298c8972d450ad" is the actual
	// hmac key, according to Larsenv.

	// also TODO: validate mlchkid with database
	hmacKey := "ce4cf29a3d6be1c2619172b5cb298c8972d450ad"

	// Parse form in preparation for finding mail.
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	for key, value := range r.PostForm {
		log.Printf("%s => %s", key, value[0])
	}

	// https://github.com/RiiConnect24/Mail-Go/wiki/check.cgi for response format
	result := "cd=100\n"
	result += "msg=Success.\n"
	result += fmt.Sprint("res=", hmacKey, "\n")
	// Random, non-zero string until we start checking
	result += fmt.Sprint("mail.flag=", RandStringBytesMaskImprSrc(33), "\n")
	result += fmt.Sprint("interval=", interval)
	w.Write([]byte(result))
}

// https://stackoverflow.com/a/31832326/3874884
var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
