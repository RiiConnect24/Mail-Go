package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// Check handles adding the proper interval for check.cgi along with future
// challenge solving and future mail existence checking.
// BUG(spotlightishere): Challenge solving isn't implemented whatsoever.
func Check(w http.ResponseWriter, r *http.Request, db *sql.DB, inter int) {
	stmt, err := db.Prepare("SELECT mlid FROM accounts WHERE mlchkid=?")
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(420, "Unable to formulate authentication statement."))
		log.Fatal(err)
		return
	}
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
	hmacKey := "ce4cf29a3d6be1c2619172b5cb298c8972d450ad"

	// Parse form in preparation for finding mail.
	err = r.ParseForm()
	if err != nil {
		fmt.Fprint(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		log.Fatal(err)
		return
	}

	//isVerified, err := Auth(r.Form)
	//if err != nil {
	//	fmt.Fprintf(w, GenNormalErrorCode(666, "Something weird happened."))
	//	log.Printf("Error checking: %v", err)
	//	return
	//} else if !isVerified {
	//	fmt.Fprintf(w, GenNormalErrorCode(220, "An authentication error occurred."))
	//	return
	//}

	mlchkid := r.Form.Get("mlchkid")
	if mlchkid == "" {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		return
	}

	// Check mlchkid
	result, err := stmt.Query(mlchkid)
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		log.Fatal(err)
		return
	}

	// By default, we'll assume there's no mail.
	// mailFlag := "0"
	resultsLoop := 0

	// Scan through returned rows.
	defer result.Close()
	for result.Next() {
		var mlid string
		err = result.Scan(&mlid)
		log.Print(mlid)
		stmt, err := db.Prepare("SELECT * FROM `mails` WHERE `recipient_id` =? AND `sent` = 0 ORDER BY `timestamp` ASC")
		if err != nil {
			log.Fatal(err)
		}
		// Splice off w from mlid
		storedMail, err := stmt.Query(mlid[1:])
		if err != nil {
			log.Fatal(err)
		}

		size := 0
		defer storedMail.Close()
		for storedMail.Next() {
			size++
		}
		err = result.Err()
		if err != nil {
			fmt.Fprintf(w, GenNormalErrorCode(420, "Unable to formulate authentication statement."))
			log.Fatal(err)
			return
		}

		// Set mail flag to number of mail taken from database
		// mailFlag = strconv.Itoa(size)
		resultsLoop++
	}

	err = result.Err()
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(420, "Unable to formulate authentication statement."))
		log.Fatal(err)
		return
	}

	/* if resultsLoop == 0 {
		// Looks like that user didn't exist.
		fmt.Fprintf(w, GenNormalErrorCode(220, "Invalid authentication.")))
		return
	} */

	// https://github.com/RiiConnect24/Mail-Go/wiki/check.cgi for response format
	fmt.Fprint(w, GenNormalErrorCode(100, "Success."),
		"res=", hmacKey, "\n",
		"mail.flag=", RandStringBytesMaskImprSrc(33), "\n",
		"interval=", interval)
}
