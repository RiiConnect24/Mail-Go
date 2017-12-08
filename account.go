package main

import (
	"database/sql"
	"net/http"
	"fmt"
	"log"
)

func Account(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")
	// TODO: figure out actual mlid generation
	stmt, err := db.Prepare("INSERT IGNORE INTO `accounts` (`mlid`,`mlchkid`, `passwd` ) VALUES (?, ?, ?)")
	if err != nil {
		w.Write([]byte(GenNormalErrorCode(450, "Database error.")))
		log.Fatal(err)
	}
	r.ParseForm()

	wiiID := r.Form.Get("mlid")
	if wiiID == "" {
		w.Write([]byte("At least humor us and use the correct syntax."))
		return
	} else if wiiID[0:1] != "w" {
		w.Write([]byte(GenNormalErrorCode(610, "Invalid Wii Friend Code.")))
		return
	} else if len(wiiID) != 17 {
		w.Write([]byte(GenNormalErrorCode(610, "Invalid Wii Friend Code.")))
		return
	}

	mlchkid := RandStringBytesMaskImprSrc(32)
	passwd := RandStringBytesMaskImprSrc(16)
	_, err = stmt.Exec(wiiID, mlchkid, passwd)

	if err != nil {
		w.Write([]byte(GenNormalErrorCode(450, "Database error.")))
		log.Fatal(err)
	} else {
		w.Write([]byte(fmt.Sprint("\n",
			GenNormalErrorCode(100, "Success."),
			"mlid=", wiiID, "\n",
			"passwd=", passwd, "\n",
			"mlchkid=", mlchkid, "\n")))
	}
}
