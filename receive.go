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

// Receive loops through stored mail and formulates a response.
// Then, if applicable, marks the mail as received.
func Receive(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Parse form.
	err := r.ParseForm()
	if err != nil {
		LogError("Error parsing form", err)
		return
	}

	isVerified, err := Auth(r.Form)
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(531, "Something weird happened."))
		LogError("Error receiving.", err)
		return
	} else if !isVerified {
		fmt.Fprintf(w, GenNormalErrorCode(230, "An authentication error occurred."))
		return
	}

	// We already know the mlid is valid from previous
	// so we don't need to further check.
	mlidWithW := r.Form.Get("mlid")
	mlid := mlidWithW[1:]

	maxsize, err := strconv.Atoi(r.Form.Get("maxsize"))
	if err != nil {
		fmt.Fprint(w, GenNormalErrorCode(330, "maxsize needs to be an int."))
		return
	}

	stmt, err := db.Prepare("SELECT * FROM `mails` WHERE `recipient_id` = ? AND `sent` = 0 ORDER BY `timestamp` ASC")
	if err != nil {
		LogError("Error preparing statement", err)
		return
	}
	storedMail, err := stmt.Query(mlid)
	if err != nil {
		LogError("Error running query against mlid", err)
		return
	}

	var totalMailOutput string
	amountOfMail := 0
	mailSize := 0

	// Statement to mark as sent once put in mail output
	updateMailState, err := db.Prepare("UPDATE `mails` SET `sent` = 1 WHERE `mail_id` = ?")
	if err != nil {
		LogError("Error preparing sent statement", err)
		return
	}

	// Loop through mail and make the output.
	wc24MimeBoundary := fmt.Sprint("BoundaryForDL", time.Now().Format("200601021504"), "/", random(1000000, 9999999))
	w.Header().Add("Content-Type", fmt.Sprint("multipart/mixed; boundary=", wc24MimeBoundary))

	defer storedMail.Close()
	for storedMail.Next() {
		// Mail is the content of the mail stored in the database.
		var mailId string
		var messageId string
		var senderWiiID string
		var mail string
		var recipientId string
		var sent int
		var timestamp string
		err = storedMail.Scan(&mailId, &messageId, &senderWiiID, &mail, &recipientId, &sent, &timestamp)
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
			break
		} else {
			totalMailOutput += individualMail
			amountOfMail++

			// Make mailSize reflect our actions.
			mailSize += len(mail)

			// We're committed at this point. Mark it that way in the db.
			_, err := updateMailState.Exec(mailId)
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
