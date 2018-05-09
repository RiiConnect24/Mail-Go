package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
)

// Delete handles delete requests of mail.
func Delete(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	stmt, err := db.Prepare("DELETE FROM `mails` WHERE `sent` = 1 AND `recipient_id` = ? ORDER BY `timestamp` ASC LIMIT ?")
	if err != nil {
		// Welp, that went downhill fast.
		fmt.Fprint(w, GenNormalErrorCode(440, "Database error."))
		LogError("Error creating delete prepared statement", err)
		return
	}

	isVerified, err := Auth(r.Form)
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(541, "Something weird happened."))
		LogError("Error parsing delete authentication", err)
		return
	} else if !isVerified {
		fmt.Fprintf(w, GenNormalErrorCode(240, "An authentication error occurred."))
		return
	}

	// We don't need to check mlid as it's been verified by Auth above.
	wiiID := r.Form.Get("mlid")

	delnum := r.Form.Get("delnum")
	actualDelnum, err := strconv.Atoi(delnum)
	if err != nil {
		fmt.Fprintf(w, GenNormalErrorCode(340, "Invalid delete value."))
		return
	}
	_, err = stmt.Exec(wiiID, actualDelnum)

	if err != nil {
		LogError("Error deleting from database", err)
		fmt.Fprint(w, GenNormalErrorCode(541, "Issue deleting mail from the database."))
	} else {
		fmt.Fprint(w, GenSuccessResponse(),
			"deletenum=", delnum)
	}
}
