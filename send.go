package main

import (
	"database/sql"
	"net/http"
	"log"
	"regexp"
	"fmt"
	"strconv"
	"bufio"
	"strings"
	"github.com/google/uuid"
	//"github.com/sendgrid/sendgrid-go"
)

var mailFormName = regexp.MustCompile(`m\d+`)
var mailFrom = regexp.MustCompile(`^MAIL FROM:\s(w[0-9]*)@(?:.*)$`)
var rcptFrom = regexp.MustCompile(`^RCPT TO:\s(.*)@(.*)$`)
var messageIDRegex = regexp.MustCompile(`Message-Id:\s<([0-9a-fA-F]*)@(?:.*)>$`)

// Send takes POSTed mail by the Wii and stores it in the database for future usage.
func Send(w http.ResponseWriter, r *http.Request, db *sql.DB, config Config) {
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")
	// Go ahead and prepare the insert statement, for later™ usage.
	stmt, err := db.Prepare("INSERT INTO `mails` (`sender_wiiID`,`mail`, `recipient_id`, `mail_id`, `message_id`) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		// Welp, that went downhill fast.
		w.Write(genErrorCode(450, "Database error."))
		return
	}

	//client := sendgrid.NewSendClient(config.SendGridKey)

	// Create maps for storage of mail.
	mailPart := make(map[string]string)

	// Parse form in preparation for finding ma	il.
	err = r.ParseMultipartForm(-1)
	if err != nil {
		log.Fatal(err)
	}

	for name, contents := range r.MultipartForm.Value {
		if mailFormName.MatchString(name) {
			log.Print("Message detected. Name: ", name)
			mailPart[name] = contents[0]
		}
	}

	// Handle the all mail! \o/
	for _, contents := range mailPart {
		var linesToRemove string
		// I'm making this a string for similar reasons as below.
		// Plus it beats repeated `strconv.Itoa`s
		var wiiRecipientIDs []string
		var pcRecipientIDs []string
		// Yes, senderID is a string. >.<
		// The database contains `w<16 digit ID>` due to previous PHP scripts.
		// POTENTIAL TODO: remove w from database?
		var messageID string
		var senderID string
		var data string

		messageID = senderID
		senderID = messageID

		// For every new line, handle as needed.
		scanner := bufio.NewScanner(strings.NewReader(contents))
		for scanner.Scan() {
			line := scanner.Text()
			// Add it to this mail's overall data.
			data += fmt.Sprintln(line)

			if line == "DATA" {
				// We don't actually need to do anything here,
				// just carry on.
				linesToRemove += fmt.Sprintln(line)
				continue
			}

			potentialMailFromWrapper := mailFrom.FindStringSubmatch(line)
			if potentialMailFromWrapper != nil {
				potentialMailFrom := potentialMailFromWrapper[1]
				if potentialMailFrom == "w9999999999990000" {
					w.Write(genErrorCode(351, "w9999999999990000 tried to send mail."))
					break
				}
				senderID = potentialMailFrom
				linesToRemove += fmt.Sprintln(line)
				continue
			}

			// -1 signifies all matches
			potentialRecipientWrapper := rcptFrom.FindAllStringSubmatch(line, -1)
			if potentialRecipientWrapper != nil {
				// We only need to work with the first match, which should be all we need.
				potentialRecipient := potentialRecipientWrapper[0]

				// layout:
				// potentialRecipient[0] = original matched string w/o groups
				// potentialRecipient[1] = w<16 digit ID>
				// potentialRecipient[2] = domain being sent to
				if potentialRecipient[2] == "wii.com" {
					// We're not gonna allow you to send to a defunct domain. ;P
				} else if potentialRecipient[2] == config.SendGridDomain {
					// Wii <-> Wii mail. We can handle this.
					wiiRecipientIDs = append(wiiRecipientIDs, potentialRecipient[1])
				} else {
					// PC <-> Wii mail. We can't handle this, but SendGrid can.
					email := fmt.Sprintf("%s@%s", potentialRecipient[1], potentialRecipient[2])
					pcRecipientIDs = append(pcRecipientIDs, email)
				}

				linesToRemove += fmt.Sprintln(line)
				continue
			}

			potentialMessageID := messageIDRegex.FindStringSubmatch(line)
			if potentialMessageID != nil {
				// We don't need to record this as it's part of DATA.
				messageID = potentialMessageID[1]
				continue
			} else {
				// We do need a message ID though.
				messageID = uuid.New().String()
				continue
			}

			w.Write(genErrorCode(351, "Your Wii sent something I couldn't understand."))
			return
		}
		if err := scanner.Err(); err != nil {
			w.Write(genErrorCode(350, "Issue iterating over strings."))
			return
		}
		mailContents := strings.Replace(data, linesToRemove, "", -1)

		// We're done figuring out the mail, now it's time to act as needed.
		// For Wii recipients, we can just insert into the database.
		for _, wiiRecipient := range wiiRecipientIDs {
			// Splice wiiRecipient to drop w from 16 digit ID.
			_, err := stmt.Exec(senderID, mailContents, wiiRecipient[1:], uuid.New().String(), messageID)
			if err != nil {
				w.Write(genErrorCode(450, "Database error."))
				return
			}
		}

		//for _ := range pcRecipientIDs {
		//
		//}
	}
}

func genErrorCode(error int, reason string) []byte {
	log.Println("[Warning] Encountered error", error, "with reason", reason)
	return []byte(fmt.Sprint(
		"cd=", strconv.Itoa(error), "\n",
		"msg=", reason, "\n"))
}
