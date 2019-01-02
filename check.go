package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"

	"github.com/RiiConnect24/Mail-Go/utilities"
)

// Check handles adding the proper interval for check.cgi along with future
// challenge solving and future mail existence checking.
func Check(c *gin.Context) {
	// Used later on for challenge solving.
	var res string

	mlchkidStmt, err := db.Prepare("SELECT `mlid` FROM accounts WHERE `mlchkid` = ?")
	if err != nil {
		ErrorResponse(c, 420, "Unable to formulate authentication statement.")
		utilities.LogError(ravenClient, "Unable to prepare check statement", err)
		return
	}
	// Grab string of interval
	interval := strconv.Itoa(global.Interval)
	// Add required headers
	c.Header("Content-Type", "text/plain;charset=utf-8")
	c.Header("X-Wii-Mail-Download-Span", interval)
	c.Header("X-Wii-Mail-Check-Span", interval)

	// Parse form in preparation for finding mail.
	mlchkid := c.PostForm("mlchkid")
	if mlchkid == "" {
		ErrorResponse(c, 320, "Unable to parse parameters.")
		return
	}

	// Grab salt + mlchkid sha512
	hashByte := sha512.Sum512(append(salt, []byte(mlchkid)...))
	hash := hex.EncodeToString(hashByte[:])

	// Check mlchkid
	result, err := mlchkidStmt.Query(hash)
	if err != nil {
		ErrorResponse(c, 320, "Unable to parse parameters.")
		utilities.LogError(ravenClient, "Unable to run mlchkid query", err)
		return
	}

	mlidStatement, err := db.Prepare("SELECT * FROM `mails` WHERE `recipient_id` =? AND `sent` = 0 ORDER BY `timestamp` ASC")
	if err != nil {
		utilities.LogError(ravenClient, "Unable to prepare mlid statement", err)
	}

	// By default, we'll assume there's no mail.
	mailFlag := "000000000000000000000000000000000"
	resultsLoop := 0
	size := 0

	// Scan through returned rows.
	defer result.Close()
	for result.Next() {
		var mlid string
		err = result.Scan(&mlid)

		key, err := hex.DecodeString("ce4cf29a3d6be1c2619172b5cb298c8972d450ad")
		if err != nil {
			utilities.LogError(ravenClient, "Unable to decode key", err)
		}

		chlng, err := hex.DecodeString(c.PostForm("chlng"))
		if err != nil {
			utilities.LogError(ravenClient, "Unable to decode chlng string", err)
		}

		h := hmac.New(sha1.New, []byte(key))
		h.Write([]byte(mlid))
		h.Write([]byte(chlng))
		res = hex.EncodeToString(h.Sum(nil))

		// Splice off w from mlid
		storedMail, err := mlidStatement.Query(mlid[1:])
		if err != nil {
			utilities.LogError(ravenClient, "Unable to run mlid", err)
			return
		}

		defer storedMail.Close()
		for storedMail.Next() {
			size++
		}
		err = result.Err()
		if err != nil {
			ErrorResponse(c, 420, "Unable to formulate authentication statement.")
			utilities.LogError(ravenClient, "Unable to get user mail", err)
			return
		}

		// Set mail flag to number of mail taken from database
		resultsLoop++
	}

	err = result.Err()
	if err != nil {
		ErrorResponse(c, 420, "Unable to formulate authentication statement.")
		utilities.LogError(ravenClient, "Generic database issue", err)
		return
	}

	if resultsLoop == 0 {
		// Looks like that user didn't exist.
		ErrorResponse(c, 220, "Invalid authentication.")
		return
	}

	if size > 0 {
		// mailFlag needs to be not one, apparently.
		// The Wii will refuse to check otherwise.
		mailFlag = utilities.RandStringBytesMaskImprSrc(33) // This isn't how Nintendo did the mail flag, how they did it is currently unknown.
	} else {
		// mailFlag was already set to 0 above.
	}

	// https://github.com/RiiConnect24/Mail-Go/wiki/check.cgi for response format
	c.String(http.StatusOK, fmt.Sprint(SuccessfulResponse,
		"res=", res, "\n",
		"mail.flag=", mailFlag, "\n",
		"interval=", interval))
}
