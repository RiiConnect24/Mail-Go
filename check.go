package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
)

var (
	// MailCheckKey is used as the basis of the SHA-1 HMAC performed for the challenge.
	MailCheckKey = []byte{0xce, 0x4c, 0xf2, 0x9a, 0x3d, 0x6b, 0xe1, 0xc2, 0x61, 0x91, 0x72, 0xb5, 0xcb, 0x29, 0x8c, 0x89, 0x72, 0xd4, 0x50, 0xad}
)

func initCheckDB() {
	var err error
	userExistsStmt, err = db.Prepare("SELECT `mlid` FROM accounts WHERE `mlchkid` = ?")
	if err != nil {
		LogError("Unable to prepare user exists statement", err)
		panic(err)
	}

	hasMailStmt, err = db.Prepare(`SELECT COUNT(mails.mail) > 0
FROM mails
WHERE mails.recipient_id = ?
AND mails.sent = 0`)

	if err != nil {
		LogError("Unable to prepare length statement", err)
		panic(err)
	}
}

var userExistsStmt *sql.Stmt
var hasMailStmt *sql.Stmt

// Check handles adding the proper interval for check.cgi along with future
// challenge solving and future mail existence checking.
func Check(w http.ResponseWriter, r *http.Request, db *sql.DB, interval string) {
	// Used later on for challenge solving.
	var res string

	// Add required headers
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")
	w.Header().Add("X-Wii-Mail-Download-Span", interval)
	w.Header().Add("X-Wii-Mail-Check-Span", interval)

	mlchkid := r.Form.Get("mlchkid")
	if mlchkid == "" {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		return
	}

	// Grab salt + mlchkid sha512, as we have this format in the database
	hashByte := sha512.Sum512(append(salt, []byte(mlchkid)...))
	hash := hex.EncodeToString(hashByte[:])

	// Check mlchkid
	var mlid string

	result := userExistsStmt.QueryRow(hash)
	err := result.Scan(&mlid)
	if err == sql.ErrNoRows {
		// Looks like that user didn't exist.
		fmt.Fprintf(w, GenNormalErrorCode(321, "User not found."))
		return
	} else if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		LogError("Unable to run check query", err)
		return
	}

	// By default, we'll assume there's no mail.
	mailFlag := "000000000000000000000000000000000"
	var hasMail bool

	// recipient_id has no w as a prefix to its mlid, so we must strip when querying.
	result = hasMailStmt.QueryRow(mlid[1:])
	err = result.Scan(&hasMail)
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to query mail availability"))
		LogError("Unable to query mail availability", err)
		return
	}

	if hasMail {
		// mailFlag needs to be not one, apparently.
		// The Wii will refuse to check otherwise.
		mailFlag = RandStringBytesMaskImprSrc(33) // This isn't how Nintendo did the mail flag, how they did it is currently unknown.
	} else {
		// mailFlag was already set to 0 above.
	}

	chlng := r.Form.Get("chlng")
	if chlng == "" {
		fmt.Fprintf(w, GenNormalErrorCode(320, "Unable to parse parameters."))
		return
	}

	h := hmac.New(sha1.New, MailCheckKey)
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
