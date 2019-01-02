package main

import (
	"github.com/RiiConnect24/Mail-Go/utilities"
	"log"
)

// Intuitive text to remind you that Mail-GO has a purging feature.
// A feature as simple as this has caused a lot of commotion.
// But fear begone, as the mailman no longer has to carry old and grotty mail.
func purgeMail() {
	// BEGONE MAIL!
	stmtIns, err := db.Prepare("DELETE FROM WC24Mail.mails WHERE `timestamp` < NOW() - INTERVAL 28 DAY;")
	if err != nil {
		utilities.LogError(ravenClient, "Failed to prepare purge statement.", err)
	}
	result, err := stmtIns.Exec()
	if err != nil {
		utilities.LogError(ravenClient, "Failed to execute purge statement.", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		utilities.LogError(ravenClient, "Failed to obtain amount of changed rows.", err)
	}

	if affected > 0 {
		log.Println("Ran purge, found", affected, " affected.")
	} else {
		log.Println("Ran purge, nothing to do.")
	}
}
