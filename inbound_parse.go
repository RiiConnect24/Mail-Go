package main

import (
        "encoding/json"
	"fmt"
	"io/ioutil"
        "strings"

	"log"
	"net/http"
	"net/mail"
	"regexp"

	"github.com/google/uuid"
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
	if potentialMailInformation == nil || potentialMailInformation[2] != global.SendGridDomain {
		log.Println("to address didn't match")
		return
	}
	// 16 digit ID
	recipientMlid := potentialMailInformation[1]

	// We "create" a response for the Wii to use, based off attachments and multipart components.
	type File struct {
		Filename string `go:"filename"`
		Charset  string `go:"charset"`
		Type     string `go:"type"`
	}

	attachmentInfo := make(map[string]File)
	err = json.Unmarshal([]byte(r.Form.Get("attachment-info")), &attachmentInfo)
	if err != nil {
                log.Printf("failed to unpack json: %v", err)
                return
	}

        hasImage := false
        hasAttachedText := false

        var attachedFile []byte

        for name, attachment := range attachmentInfo {
	    attachmentData, _, err := r.FormFile(name)
	    if err == http.ErrMissingFile {
		// We don't care if there's nothing, it'll just stay nil.
	    } else if err != nil {
	    	log.Printf("failed to read attachment from form: %v", err)
	  	return
	    } else {
                if strings.Contains(attachment.Type,  "image") && hasImage == false {
	    	    attachedFile, err = ioutil.ReadAll(attachmentData)
              	    if err != nil {
	    	    	log.Printf("failed to read image attachment from form: %v", err)
	    	    	return
	    	    }
                    hasImage = true
                } else if strings.Contains(attachment.Type, "text") && hasAttachedText == false && text == "No message provided." {
                    attachedText, err := ioutil.ReadAll(attachmentData)
                    text = string(attachedText)
                    if err != nil {
                        log.Printf("failed to read text attachment from form: %v", err)
                        return
                    }
                    hasAttachedText = true
                }
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

	if global.Datadog {
		err := dataDogClient.Incr("mail.received_mail_sendgrid", nil, 1)
		if err != nil {
			panic(err)
		}
	}

	fmt.Fprint(w, "thanks sendgrid")
}
