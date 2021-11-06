package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/smtp"
	"regexp"
	"strings"
)

func initSendDB() {
	var err error
	mailInsertStmt, err = db.Prepare("INSERT INTO `mails` (`sender_wiiID`,`mail`, `recipient_id`, `mail_id`, `message_id`) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		LogError("Error preparing mail insertion statement", err)
		panic(err)
	}

	accountExistsStmt, err = db.Prepare("SELECT EXISTS(SELECT 1 FROM `accounts` WHERE `mlid` = ?)")
	if err != nil {
		LogError("Error preparing account existence statement", err)
		panic(err)
	}
}

var mailInsertStmt *sql.Stmt
var accountExistsStmt *sql.Stmt

var mailFormName = regexp.MustCompile(`m\d+`)
var mailFrom = regexp.MustCompile(`^MAIL FROM:\s(.*)@(?:.*)$`)
var mailFrom2 = regexp.MustCompile(`^From:\s(.*)@(?:.*)$`)
var rcptFrom = regexp.MustCompile(`^RCPT TO:\s(.*)@(.*)$`)

// Send takes POSTed mail by the Wii and stores it in the database for future usage.
func Send(w http.ResponseWriter, r *http.Request, db *sql.DB, config Config) {
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")

	// Create maps for storage of mail.
	mailPart := make(map[string]string)

	// Parse form in preparation for finding mail.
	err := r.ParseMultipartForm(-1)
	if err != nil {
		fmt.Fprint(w, GenNormalErrorCode(350, "Failed to parse mail."))
		LogError("Failed to parse mail", err)
		return
	}

	// Now check if it can be verified
	isVerified, authedWiiId, err := Auth(r.Form)
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(551, "Something weird happened."))
		LogError("Error changing from authentication database.", err)
		return
	} else if !isVerified {
		fmt.Fprintf(w, GenNormalErrorCode(250, "An authentication error occurred."))
		return
	}

	for name, contents := range r.MultipartForm.Value {
		if mailFormName.MatchString(name) {
			mailPart[name] = contents[0]
		}
	}

	eventualOutput := GenSuccessResponse()
	eventualOutput += fmt.Sprint("mlnum=", len(mailPart), "\n")

	// Handle all the mail! \o/
	for mailNumber, contents := range mailPart {
		var linesToRemove string
		// I'm making this a string for similar reasons as below.
		// Plus it beats repeated `strconv.Itoa`s
		var wiiRecipientIDs []string
		var pcRecipientIDs []string
		// Yes, senderID is a string. >.<
		// The database contains `w<16 digit ID>` due to previous PHP scripts.
		// POTENTIAL TODO: remove w from database?
		var senderID string
		var data string

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
				// Ensure MAIL FROM matches the authed mlid (#29)
				if potentialMailFrom != authedWiiId {
					eventualOutput += GenMailErrorCode(mailNumber, 351, "Attempt to impersonate another user.")
					break
				} else if potentialMailFrom == "w9999999900000000" {
					eventualOutput += GenMailErrorCode(mailNumber, 351, "w9999999900000000 tried to send mail.")
					break
				}
				senderID = potentialMailFrom
				linesToRemove += fmt.Sprintln(line)
				continue
			}

			potentialMailFromWrapper2 := mailFrom2.FindStringSubmatch(line)
			if potentialMailFromWrapper2 != nil {
				potentialMailFrom2 := potentialMailFromWrapper2[1]
				// Ensure From matches the authed mlid (#29)
				if potentialMailFrom2 != authedWiiId {
					eventualOutput += GenMailErrorCode(mailNumber, 351, "Attempt to impersonate another user.")
					return
				} else if potentialMailFrom2 == "w9999999900000000" {
					eventualOutput += GenMailErrorCode(mailNumber, 351, "w9999999900000000 tried to send mail.")
					return
				}
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
			}
		}
		if err := scanner.Err(); err != nil {
			eventualOutput += GenMailErrorCode(mailNumber, 551, "Issue iterating over strings.")
			LogError("Error reading from scanner", err)
			return
		}
		mailContents := strings.Replace(data, linesToRemove, "", -1)
		// Replace all @wii.com references in the
		// friend request email with our own domain.
		// Format: w9004342343324713@wii.com <mailto:w9004342343324713@wii.com>
		mailContents = strings.Replace(mailContents,
			fmt.Sprintf("%s@wii.com <mailto:%s@wii.com>", senderID, senderID),
			fmt.Sprintf("%s@%s <mailto:%s@%s>", senderID, global.SendGridDomain, senderID, global.SendGridDomain),
			-1)

		// We're done figuring out the mail, now it's time to act as needed.
		// For Wii recipients, we can just insert into the database.
		i := 0
		for _, wiiRecipient := range wiiRecipientIDs {
			if i > 10 {
				continue
			}

			// Check that the account actually exists (#15)
			var exists bool
			existsErr := accountExistsStmt.QueryRow(wiiRecipient).Scan(&exists)
			if existsErr != nil && existsErr != sql.ErrNoRows {
				eventualOutput += GenMailErrorCode(mailNumber, 551, "Issue verifying recipients.")
				LogError("Error verifying recipient account existence", err)
				return
			} else if !exists {
				// Account doesn't exist, ignore
				continue
			}

			// Splice wiiRecipient to drop w from 16 digit ID.
			_, err := mailInsertStmt.Exec(senderID, mailContents, wiiRecipient[1:], uuid.New().String(), uuid.New().String())
			if err != nil {
				eventualOutput += GenMailErrorCode(mailNumber, 450, "Database error.")
				LogError("Error inserting mail", err)
				return
			}

			i += 1
		}

		i = 0
		for _, pcRecipient := range pcRecipientIDs {
			if i > 10 {
				continue
			}

			err := handlePCmail(config, senderID, pcRecipient, mailContents)
			if err != nil {
				LogError("Error sending mail via SendGrid", err)
				eventualOutput += GenMailErrorCode(mailNumber, 551, "Issue sending mail via SendGrid.")
				return
			}

			if pcRecipient == "trigger@applet.ifttt.com" {
				// Send dummy confirmation mail for IFTTT.
				var iftttMail string

				iftttMail = "From: trigger@applet.ifttt.com\n"
				iftttMail += "Subject: Trigger\n"
				iftttMail += "To: " + senderID + "@rc24.xyz\n"
				iftttMail += "MIME-Version: 1.0\n"
				iftttMail += "Content-Type: MULTIPART/mixed; BOUNDARY=\"ifttt\"\n"
				iftttMail += "--ifttt\n"
				iftttMail += "Content-Type: TEXT/plain; CHARSET=utf-8\n"
				iftttMail += "Content-Description: wiimail\n\n"
				iftttMail += "Trigger has been ran!\n\n"
				// iftttMail += "--ifttt--"

				_, err := mailInsertStmt.Exec("trigger@applet.ifttt.com", iftttMail, senderID[1:], uuid.New().String(), uuid.New().String())
				if err != nil {
					eventualOutput += GenMailErrorCode(mailNumber, 450, "Database error.")
					LogError("Error inserting mail", err)
					return
				}
			}

			i += 1
		}
		eventualOutput += GenMailErrorCode(mailNumber, 100, "Success.")

		if global.Datadog {
			err := dataDogClient.Incr("mail.sent_mail", nil, 1.0)
			if err != nil {
				panic(err)
			}
		}
	}

	// We're completely done now.
	fmt.Fprint(w, eventualOutput)
}

func handlePCmail(config Config, senderID string, pcRecipient string, mailContents string) error {
	// Connect to the remote SMTP server.
	host := "smtp.sendgrid.net"
	auth := smtp.PlainAuth(
		"",
		"apikey",
		config.SendGridKey,
		host,
	)
	// The only reason we can get away with the following is
	// because the Wii POSTs valid SMTP syntax.
	return smtp.SendMail(
		fmt.Sprint(host, ":587"),
		auth,
		fmt.Sprintf("%s@%s", senderID, config.SendGridDomain),
		[]string{pcRecipient},
		[]byte(mailContents),
	)

}
