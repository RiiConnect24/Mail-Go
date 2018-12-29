package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"

	"github.com/RiiConnect24/Mail-Go/utilities"
)

// Delete handles delete requests of mail.
func Delete(c *gin.Context) {
	stmt, err := db.Prepare("DELETE FROM `mails` WHERE `sent` = 1 AND `recipient_id` = ? ORDER BY `timestamp` ASC LIMIT ?")
	if err != nil {
		// Welp, that went downhill fast.
		ErrorResponse(c, 440, "Database error.")
		utilities.LogError(ravenClient, "Error creating delete prepared statement", err)
		return
	}

	wiiID := c.PostForm("mlid")
	isVerified, err := Auth(wiiID, c.PostForm("passwd"))
	if err != nil {
		ErrorResponse(c, 541, "Something weird happened.")
		utilities.LogError(ravenClient, "Error parsing delete authentication", err)
		return
	} else if !isVerified {
		ErrorResponse(c, 240, "An authentication error occurred.")
		return
	}

	// We don't need to check mlid as it's been verified by Auth above.
	delnum := c.PostForm("delnum")
	actualDelnum, err := strconv.Atoi(delnum)
	if err != nil {
		ErrorResponse(c, 340, "Invalid delete value.")
		return
	}
	_, err = stmt.Exec(wiiID, actualDelnum)

	if err != nil {
		utilities.LogError(ravenClient, "Error deleting from database", err)
		ErrorResponse(c, 220, "Issue deleting mail from the database.")
	} else {
		c.String(http.StatusOK,
			SuccessfulResponse+
				"deletenum=", delnum)
	}
}