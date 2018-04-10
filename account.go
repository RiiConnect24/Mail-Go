package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/go-sql-driver/mysql"
)

func Account(w http.ResponseWriter, r *http.Request, db *sql.DB, mode int) {
	w.Header().Add("Content-Type", "text/plain;charset=utf-8")
	// TODO: figure out actual mlid generation

	r.ParseForm()

	wiiID := r.Form.Get("mlid")
	if wiiID == "" {
		w.Write([]byte(GenNormalErrorCode(310, "At least humor us and use the correct syntax.")))
		return
	} else if wiiID[0:1] != "w" {
		w.Write([]byte(GenNormalErrorCode(310, "Invalid Wii Friend Code.")))
		return
	} else if len(wiiID) != 17 {
		w.Write([]byte(GenNormalErrorCode(310, "Invalid Wii Friend Code.")))
		return
	}

	// mode 0: account.cgi call (do not insert into database, only generate mlchkid and passwd)
	// mode 1: check.cgi call
	// mode 2: send.cgi call or receive.cgi call

	if mode == 0 {
		mlchkid := RandStringBytesMaskImprSrc(32)
		passwd := RandStringBytesMaskImprSrc(16)

		w.Write([]byte(fmt.Sprint("\n",
			GenNormalErrorCode(100, "Success."),
			"mlid=", wiiID, "\n",
			"passwd=", passwd, "\n",
			"mlchkid=", mlchkid, "\n")))
	} else if mode > 0 {
		if mode == 1 {
			stmt, err := db.Prepare("INSERT IGNORE INTO `accounts` (`mlid`, `mlchkid`, `passwd` ) VALUES (?, ?, ?)")
			defer stmt.Close()

			mlchkid, err := bcrypt.GenerateFromPassword([]byte(r.Form.Get("mlchkid")), bcrypt.DefaultCost)
			passwd := ""

			_, err = stmt.Exec(wiiID, mlchkid, passwd)
			if err != nil {
				GenNormalErrorCode(410, "Database error.")
				log.Fatal(err)
			}
		} else if mode == 2 {
			stmt, err := db.Prepare("UPDATE `accounts` SET `passwd` = ? WHERE `mlid` = ?")
			defer stmt.Close()

			if err == sql.ErrNoRows {
				w.Write([]byte("Not registered yet.")) // Replace
			}

			passwd, err := bcrypt.GenerateFromPassword([]byte(r.Form.Get("passwd")), bcrypt.DefaultCost)

			_, err = stmt.Exec(passwd, wiiID)
			if err != nil {
				GenNormalErrorCode(410, "Database error.")
				log.Fatal(err)
			}
		}
	}
}
