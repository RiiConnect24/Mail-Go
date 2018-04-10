package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

// Delete handles delete requests of mail.
func Delete(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	stmt, err := db.Prepare("DELETE FROM `mails` WHERE `sent` = 1 AND `recipient_id` = ? ORDER BY `timestamp` ASC LIMIT ?")
	if err != nil {
		// Welp, that went downhill fast.
		w.Write([]byte(GenNormalErrorCode(440, "Database error.")))
		log.Fatal(err)
	}
	r.ParseForm()

	auth := Auth(w, r, 2)

	if auth == 3 {
		w.Write([]byte(GenNormalErrorCode(240, "An authentication error occurred.")))
		return
	}

	wiiID := r.Form.Get("mlid")
	delnum := r.Form.Get("delnum")
	_, err = stmt.Exec(wiiID, delnum)

	if err != nil {
		log.Fatal(err)
		w.Write([]byte(fmt.Sprint(GenNormalErrorCode(541, "Issue deleting mail from the database."))))
	} else {
		w.Write([]byte(fmt.Sprint(GenNormalErrorCode(100, "Success."),
			"delnum=", delnum)))
	}
}
