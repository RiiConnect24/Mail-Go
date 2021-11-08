package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
)

func initCheckDB() {
	var err error
	mlchkidStmt, err = db.Prepare("SELECT `mlid` FROM accounts WHERE `mlchkid` = ?")
	if err != nil {
		LogError("Unable to prepare mlchkid statement", err)
		panic(err)
	}

	mlidStatement, err = db.Prepare("SELECT * FROM `mails` WHERE `recipient_id` =? AND `sent` = 0 ORDER BY `timestamp` ASC")
	if err != nil {
		LogError("Unable to prepare mlid statement", err)
		panic(err)
	}
}

var mlchkidStmt *sql.Stmt
var mlidStatement *sql.Stmt

// Check handles adding the proper interval for check.cgi along with future
// challenge solving and future mail existence checking.
func Check(w http.ResponseWriter, r *http.Request, db *sql.DB, inter int) {
	// Used later on for challenge solving.
	var res string

	// Grab string of interval
	interval := strconv.Itoa(inter)
	// Add required headers
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")
	w.Header().Add("X-Wii-Mail-Download-Span", interval)
	w.Header().Add("X-Wii-Mail-Check-Span", interval)

	// Parse form in preparation for finding mail.
	err := r.ParseForm()
	if err != nil {
		fmt.Fprint(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		LogError("Unable to parse form", err)
		return
	}

	mlchkid := r.Form.Get("mlchkid")
	if mlchkid == "" {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		return
	}

	// Grab salt + mlchkid sha512
	hashByte := sha512.Sum512(append(salt, []byte(mlchkid)...))
	hash := hex.EncodeToString(hashByte[:])

	// Check mlchkid
	result, err := mlchkidStmt.Query(hash)
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		LogError("Unable to run mlchkid query", err)
		return
	}

	// By default, we'll assume there's no mail.
	mailFlag := "000000000000000000000000000000000"
	resultsLoop := 0
	size := 0

	var mlid string

	// Scan through returned rows.
	defer result.Close()
	for result.Next() {
		err = result.Scan(&mlid)
		if err != nil {
			fmt.Fprintf(w, GenNormalErrorCode(420, "Unable to formulate authentication statement."))
			LogError("Unable to run mlid", err)
			return
		}

		// Splice off w from mlid
		storedMail, err := mlidStatement.Query(mlid[1:])
		if err != nil {
			fmt.Fprintf(w, GenNormalErrorCode(420, "Unable to formulate authentication statement."))
			LogError("Unable to run mlid", err)
			return
		}

		defer storedMail.Close()
		for storedMail.Next() {
			size++
		}
		err = result.Err()
		if err != nil {
			fmt.Fprintf(w, GenNormalErrorCode(420, "Unable to formulate authentication statement."))
			LogError("Unable to get user mail", err)
			return
		}

		// Set mail flag to number of mail taken from database
		resultsLoop++
	}

	if size > 0 {
		// mailFlag needs to be not one, apparently.
		// The Wii will refuse to check otherwise.
		mailFlag = RandStringBytesMaskImprSrc(33) // This isn't how Nintendo did the mail flag, how they did it is currently unknown.
	} else {
		// mailFlag was already set to 0 above.
	}

	key, err := hex.DecodeString("ce4cf29a3d6be1c2619172b5cb298c8972d450ad")
	if err != nil {
		LogError("Unable to decode key", err)
	}

	chlng := r.Form.Get("chlng")
	if chlng == "" {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		return
	}

	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(chlng))
	h.Write([]byte("\n"))
	h.Write([]byte(mlid))
	h.Write([]byte("\n"))
	h.Write([]byte(mailFlag))
	h.Write([]byte("\n"))
	h.Write([]byte(interval))
	res = hex.EncodeToString(h.Sum(nil))

	err = result.Err()
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(420, "Unable to formulate authentication statement."))
		LogError("Generic database issue", err)
		return
	}

	if resultsLoop == 0 {
		// Looks like that user didn't exist.
		fmt.Fprintf(w, GenNormalErrorCode(321, "User not found."))
		return
	}

	if global.Datadog {
		err := dataDogClient.Incr("mail.checked", nil, 1)
		if err != nil {
			panic(err)
		}
	}

	// https://github.com/RiiConnect24/Mail-Go/wiki/check.cgi for response format
	fmt.Fprint(w, GenSuccessResponse(),
		"res=", res, "\n",
		"mail.flag=", mailFlag, "\n",
		"interval=", interval)
}
