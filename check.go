package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// Check handles adding the proper interval for check.cgi along with future
// challenge solving and future mail existence checking.
// BUG(spotlightishere): Challenge solving isn't implemented whatsoever,
// nor is if mail even exists.
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

	// https://github.com/RiiConnect24/Mail-Go/wiki/check.cgi for response format
	result := GenNormalErrorCode(100, "Success.")
	result += fmt.Sprint("res=", hmacKey, "\n")
	// Random, non-zero string until we start checking
	result += fmt.Sprint("mail.flag=", RandStringBytesMaskImprSrc(33), "\n")
	result += fmt.Sprint("interval=", interval)
	w.Write([]byte(result))
}