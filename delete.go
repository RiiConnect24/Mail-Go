package main

import (
	"database/sql"
	"net/http"
	"log"
	"fmt"
)

func Delete(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	stmt, err := db.Prepare("DELETE FROM `mails` WHERE `sent` = 1 AND `recipient_id` = ? ORDER BY `timestamp` ASC LIMIT ?")
	if err != nil {
		log.Fatal(stmt)
	}
	r.ParseForm()

	wiiID := r.Form.Get("mlid")[1:]
	delnum := r.Form.Get("delnum")
	log.Fatal(stmt.Exec(wiiID, delnum))

	if stmt != nil {
		w.Write([]byte(fmt.Sprint("cd=100\n",
			"msg=Success.\n",
			"delnum=", delnum)))
	}
}
