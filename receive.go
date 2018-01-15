package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Receive loops through stored mail and formulates a response.
// Then, if applicable, marks the mail as received.
func Receive(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Parse form.
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	// Go expects multiple values for a key.
	// We take the first, and then we use
	// splicing to take char 1 -> end
	mlidWithW := r.Form.Get("mlid")
	if len(mlidWithW) != 17 {
		// 17 is size of 4 x 4 digits plus starting W
		w.Write([]byte("If you're gonna try and interface with this script, at least have mlid the proper length."))
		return
	}
	mlid := mlidWithW[1:]

	maxsize, err := strconv.Atoi(r.Form.Get("maxsize"))
	if err != nil {
		w.Write([]byte("maxsize needs to be an int."))
		return
	}

	stmt, err := db.Prepare("SELECT * FROM `mails` WHERE `recipient_id` =? AND `sent` = 0 ORDER BY `timestamp` ASC")
	if err != nil {
		log.Fatal(err)
	}
	storedMail, err := stmt.Query(mlid)
	if err != nil {
		log.Fatal(err)
	}

	var totalMailOutput string
	amountOfMail := 0
	mailSize := 0

	// Statement to mark as sent once put in mail output
	updateMailState, err := db.Prepare("UPDATE `mails` SET `sent` = 1 WHERE `mail_id` =?")
	if err != nil {
		log.Fatal(err)
	}

	// Loop through mail and make the output.
	wc24MimeBoundary := fmt.Sprint("BoundaryForDL", fmt.Sprint(time.Now().Format("200601021504")), "/", random(1000000, 9999999))
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
		mail := strings.ReplaceAll(mail, "\n", "\r\n", -1)
		individualMail += mail

		// Don't add if the mail would exceed max size.
		if (len(totalMailOutput) + len(individualMail)) > maxsize {
			break
		} else {
			totalMailOutput += individualMail
			amountOfMail++

			// Make mailSize reflect our actions.
			mailSize = len(totalMailOutput)

			// We're committed at this point. Mark it that way in the db.
			_, err := updateMailState.Exec(mailId)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Make sure nothing failed.
	err = storedMail.Err()
	if err != nil {
		panic(err)
	}

	w.Header().Add("Content-Type", fmt.Sprint("multipart/mixed; boundary=", wc24MimeBoundary))
	request := fmt.Sprint("--", wc24MimeBoundary, "\r\n",
		"Content-Type: text/plain\r\n\r\n",
		"This part is ignored.\r\n\r\n\r\n\n",
		"cd=100\n",
		"msg=Success.\n",
		"mailnum=", amountOfMail, "\n",
		"mailsize=", mailSize, "\n",
		"allnum=", amountOfMail, "\n",
		totalMailOutput,
		"\r\n--", wc24MimeBoundary, "--\r\n")
	w.Write([]byte(request))
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
