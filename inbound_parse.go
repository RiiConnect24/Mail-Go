package main

import (
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"regexp"
)

var mailDomain *regexp.Regexp

func sendGridHandler(w http.ResponseWriter, r *http.Request) {
	// We sincerely hope someone won't attempt to send more than a 11MB image.
	// but, if they do, now they have 10mb for image and 1mb for text + etc
	// (still probably too much)
	err := r.ParseMultipartForm(-1)
	if err != nil {
		log.Printf("Unable to parse form: %v", err)
		return
	}

        text := r.Form.Get("text")

	// TODO: Properly verify attachments.
	if r.Form.Get("from") == "" || r.Form.Get("to") == "" {
		// something was nil
		log.Println("Something happened to SendGrid... is someone else accessing?")
		return
	}

        // If there's no text in the email.
        if text == "" {
                text = "No message provided."
        }

	// Figure out who sent it.
	fromAddress, err := mail.ParseAddress(r.Form.Get("from"))
	if err != nil {
		log.Printf("given from address is invalid: %v", err)
		return
	}

	toAddress := r.Form.Get("to")
	// Validate who's being mailed.
	potentialMailInformation := mailDomain.FindStringSubmatch(toAddress)
	log.Println(potentialMailInformation)
	if potentialMailInformation == nil || potentialMailInformation[2] != global.SendGridDomain {
		log.Println("to address didn't match")
		return
	}
	// 16 digit ID
	recipientMlid := potentialMailInformation[1]

	// We "create" a response for the Wii to use, based off attachments and multipart components.
	// TODO: potentially handle all attachments until first image type?
	var attachedFile []byte
	attachment, _, err := r.FormFile("attachment1")
	if err == http.ErrMissingFile {
		// We don't care if there's nothing, it'll just stay nil.
	} else if err != nil {
		log.Printf("failed to read attachment from form: %v", err)
		return
	} else {
		attachedFile, err = ioutil.ReadAll(attachment)
		if err != nil {
			log.Printf("failed to read attachment from form: %v", err)
			return
		}
	}

	wiiMail, err := FormulateMail(fromAddress.Address, toAddress, r.Form.Get("subject"), text, attachedFile)
	if err != nil {
		log.Printf("error formulating mail: %v", err)
		return
	}

	// On a normal Wii service, we'd return the cd/msg response.
	// This goes to SendGrid, and we hope the database error is resolved
	// later on - any non-success tells it to POST again.
	stmt, err := db.Prepare("INSERT INTO `mails` (`sender_wiiID`,`mail`, `recipient_id`, `mail_id`, `message_id`) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Printf("Database error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = stmt.Exec(fromAddress.Address, wiiMail, recipientMlid, uuid.New().String(), uuid.New().String())
	if err != nil {
		log.Printf("Database error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "thanks sendgrid")
}
