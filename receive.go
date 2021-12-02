package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func initReceiveDB() {
	var err error
	getReceiveStmt, err = db.Prepare("SELECT mail_id, mail FROM mails WHERE recipient_id = ? AND sent = 0 ORDER BY timestamp ASC")
	if err != nil {
		LogError("Error preparing mail retrieval statement", err)
		panic(err)
	}

	// Statement to mark as sent once put in mail output
	updateMailStateStmt, err = db.Prepare("UPDATE mails SET sent = 1 WHERE mail_id = ?")
	if err != nil {
		LogError("Error preparing mail state update statement", err)
		panic(err)
	}
}

var getReceiveStmt *sql.Stmt
var updateMailStateStmt *sql.Stmt

// Receive loops through stored mail and formulates a response.
// Then, if applicable, marks the mail as received.
func Receive(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	mlid := r.Form.Get("mlid")
	passwd := r.Form.Get("passwd")

	err := checkPasswdValidity(mlid, passwd)
	if err == ErrInvalidCredentials {
		fmt.Fprintf(w, GenNormalErrorCode(230, "An authentication error occurred."))
		return
	} else if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(531, "Something weird happened."))
		LogError("Error receiving.", err)
		return
	}

	maxsize, err := strconv.Atoi(r.Form.Get("maxsize"))
	if err != nil {
		fmt.Fprint(w, GenNormalErrorCode(330, "maxsize needs to be an int."))
		return
	}

	// We must strip the first w from the received mlid as the database stores it without.
	storedMail, err := getReceiveStmt.Query(mlid[1:])
	if err != nil {
		LogError("Error running query against mlid", err)
		return
	}

	var totalMailOutput string
	amountOfMail := 0
	mailSize := 0

	// Loop through mail and make the output.
	wc24MimeBoundary := GenerateBoundary()
	w.Header().Add("Content-Type", fmt.Sprint("multipart/mixed; boundary=", wc24MimeBoundary))

	defer storedMail.Close()
	for storedMail.Next() {
		// Mail is the content of the mail stored in the database.
		var mailId string
		var mail string
		err = storedMail.Scan(&mailId, &mail)
		if err != nil {
			// Hopefully not, but make sure the row layout is the same.
			panic(err)
		}
		individualMail := fmt.Sprint("\r\n--", wc24MimeBoundary, "\r\n")
		individualMail += "Content-Type: text/plain\r\n\r\n"

		// In the RiiConnect24 database, some mail use CRLF
		// instead of a Unix newline.
		// We go ahead and remove this from the mail
		// in order to not confuse the Wii.
		// BUG(larsenv): make the database not do this
		mail = strings.Replace(mail, "\n", "\r\n", -1)
		mail = strings.Replace(mail, "\r\r\n", "\r\n", -1)
		individualMail += mail

		// Don't add if the mail would exceed max size.
		if (len(totalMailOutput) + len(individualMail)) > maxsize {
			storedMail.Close()
			break
		} else {
			totalMailOutput += individualMail
			amountOfMail++

			// Make mailSize reflect our actions.
			mailSize += len(mail)

			// We're committed at this point. Mark it that way in the db.
			_, err := updateMailStateStmt.Exec(mailId)
			if err != nil {
				LogError("Unable to mark mail as sent", err)
			}
		}
	}

	// Make sure nothing failed.
	err = storedMail.Err()
	if err != nil {
		LogError("General database error", err)
	}

	if global.Datadog {
		err := dataDogClient.Incr("mail.received_mail", nil, float64(amountOfMail))
		if err != nil {
			LogError("Unable to update received_mail.", err)
		}
	}

	request := fmt.Sprint("--", wc24MimeBoundary, "\r\n",
		"Content-Type: text/plain\r\n\r\n",
		"This part is ignored.\r\n\r\n\r\n\n",
		GenSuccessResponse(),
		"mailnum=", amountOfMail, "\n",
		"mailsize=", mailSize, "\n",
		"allnum=", amountOfMail, "\n",
		totalMailOutput,
		"\r\n--", wc24MimeBoundary, "--\r\n")
	fmt.Fprint(w, request)
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
