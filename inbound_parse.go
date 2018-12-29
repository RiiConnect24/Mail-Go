package main

import (
	"github.com/RiiConnect24/Mail-Go/utilities"
	"github.com/gin-gonic/gin"
	"io/ioutil"

	"log"
	"net/http"
	"net/mail"
	"regexp"

	"github.com/google/uuid"
)

var mailDomain *regexp.Regexp

func sendGridHandler(c *gin.Context) {
	text := c.PostForm("text")

	// TODO: Properly verify attachments.
	if c.PostForm("from") == "" || c.PostForm("to") == "" {
		// something was nil
		log.Println("Something happened to SendGrid... is someone else accessing?")
		return
	}

	// If there's no text in the email.
	if text == "" {
		text = "No message provided."
	}

	// Figure out who sent it.
	fromAddress, err := mail.ParseAddress(c.PostForm("from"))
	if err != nil {
		log.Printf("given from address is invalid: %v", err)
		return
	}

	toAddress := c.PostForm("to")
	// Validate who's being mailed.
	potentialMailInformation := mailDomain.FindStringSubmatch(toAddress)
	if potentialMailInformation == nil || potentialMailInformation[2] != global.SendGridDomain {
		log.Println("to address didn't match")
		return
	}
	// 16 digit ID
	recipientMlid := potentialMailInformation[1]

	// We "create" a response for the Wii to use, based off attachments and multipart components.
	// TODO: potentially handle all attachments until first image type?
	var attachedFile []byte
	attachment, err := c.FormFile("attachment1")
	if err == http.ErrMissingFile {
		// We don't care if there's nothing, it'll just stay nil.
	} else if err != nil {
		utilities.LogError(ravenClient, "Failed to read attachment from form.", err)
		c.Status(http.StatusInternalServerError)
		return
	} else {
		file, err := attachment.Open()
		if err != nil {
			utilities.LogError(ravenClient, "Failed to open attachment from form.", err)
			c.Status(http.StatusInternalServerError)
		}
		attachedFile, err = ioutil.ReadAll(file)
		if err != nil {
			utilities.LogError(ravenClient, "Failed to read attachment from form.", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	wiiMail, err := FormulateMail(fromAddress.Address, toAddress, c.PostForm("subject"), text, attachedFile)
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
		c.Status(http.StatusInternalServerError)
		return
	}

	_, err = stmt.Exec(fromAddress.Address, wiiMail, recipientMlid, uuid.New().String(), uuid.New().String())
	if err != nil {
		log.Printf("Database error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, "thanks sendgrid")
}