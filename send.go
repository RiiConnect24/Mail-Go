package main

import (
	"database/sql"
	"net/http"
	//"github.com/google/uuid"
	"log"
	"regexp"
	"fmt"
	"strconv"
	"bufio"
	"strings"
	"github.com/google/uuid"
)

var mailFrom = regexp.MustCompile(`^MAIL FROM:\s(w[0-9]*)@(?:.*)$`)
var rcptFrom = regexp.MustCompile(`^RCPT TO:\s(.*)mails@(.*)$`)
var messageIDRegex = regexp.MustCompile(`Message-Id:\s<([0-9a-fA-F]*)@(?:.*)>$`)
var dataRegex = regexp.MustCompile(`^DATA$`)

// Send takes POSTed mail by the Wii and stores it in the database for future usage.
func Send(w http.ResponseWriter, r *http.Request, db *sql.DB, config Config) {
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")

	// Create maps for storage of mail.
	mailPart := make(map[string]string)

	// Parse form in preparation for finding mail.
	err := r.ParseMultipartForm(-1)
	if err != nil {
		log.Fatal(err)
	}

	for name, contents := range r.MultipartForm.Value {
		if name[0] == 'm' {
			log.Print("Message detected. :robot:\nName:", name)
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

		// For every new line, handle as needed.
		scanner := bufio.NewScanner(strings.NewReader(contents))
		for scanner.Scan() {
			line := scanner.Text()
			potentialMailFrom := mailFrom.FindString(line)
			if potentialMailFrom != "" {
				if potentialMailFrom == "w9999999999990000" {
					w.Write(genErrorCode(351, "w9999999999990000 tried to send mail."))
					break
				}
				senderID = potentialMailFrom
				linesToRemove += fmt.Sprintln(line)
				continue
			}

			// -1 signifies all matches
			potentialRecipient := rcptFrom.FindAllStringSubmatch(line, -1)[0]
			if potentialRecipient != nil {
				// layout:
				// potentialRecipient[0] = w<16 digit ID>
				// potentialRecipient[1] = domain being sent to
				if potentialRecipient[1] == "wii.com" {
					// We're not gonna allow you to send to a defunct domain.. ;P
					// Add it to be removed anyway.
					linesToRemove += fmt.Sprintln(line)
					continue
				} else if potentialRecipient[1] == config.SendGridDomain {
					// Wii <-> Wii mail. We can handle this.
					wiiRecipientIDs = append(wiiRecipientIDs, potentialRecipient[0][1:])
				} else {
					// PC <-> Wii mail. We can't handle this, but SendGrid can.
					pcRecipientIDs = append(pcRecipientIDs, potentialRecipient[0][1:])
				}

				linesToRemove += fmt.Sprintln(line)
				continue
			}

			potentialMessageID := messageIDRegex.FindString(line)
			if potentialMessageID != "" {
				messageID = potentialMessageID
				linesToRemove += fmt.Sprintln(line)
				continue
			} else {
				// We do need a message ID though.
				messageID = uuid.New().String()
			}

			potentialDataMessage := dataRegex.FindString(line)
			if potentialDataMessage != "" {
				// Party's over. Go home.
				break
			}

			w.Write(genErrorCode(420, "Your Wii sent something I couldn't understand."))
		}
		if err := scanner.Err(); err != nil {
			w.Write(genErrorCode(666, "Issue iterating over strings."))
		}


		// We're done figuring out the mail, now it's time to act as needed.
		
	}
}

func genErrorCode(error int, reason string) []byte {
	log.Println("[Warning] Encountered error ", error, " with reason ", reason)
	return []byte(fmt.Sprint(
		"cd=", strconv.Itoa(error), "\n",
		"msg=", reason, "\n"))
}

func handleSendGrid() {

}
