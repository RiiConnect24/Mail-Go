package main

import (
	"fmt"
	"log"
	"net/http"
	_ "github.com/go-sql-driver/mysql"
)

func Account(w http.ResponseWriter, r *http.Request) {
	wiiID := r.Form.Get("mlid")
	if !friendCodeIsValid(wiiID) {
		fmt.Fprint(w, GenNormalErrorCode(610, "Invalid Wii Friend Code."))
	}

	w.Header().Add("Content-Type", "text/plain;charset=utf-8")

	stmt, err := db.Prepare("INSERT IGNORE INTO `accounts` (`mlid`,`mlchkid`, `passwd` ) VALUES ('?', '?', '?')")
	if err != nil {
		fmt.Fprint(w, GenNormalErrorCode(450, "Database error."))
		log.Fatal(err)
		return
	}

	mlchkid := RandStringBytesMaskImprSrc(32)
	passwd := RandStringBytesMaskImprSrc(16)

	_, err = stmt.Exec(wiiID, mlchkid, passwd)
	if err != nil {
		fmt.Fprint(w, GenNormalErrorCode(450, "Database error."))
		log.Println(err)
		return
	}

	fmt.Fprint(w, "\n",
		GenNormalErrorCode(100, "Success."),
		"mlid=", wiiID, "\n",
		"passwd=", passwd, "\n",
		"mlchkid=", mlchkid, "\n")
}
