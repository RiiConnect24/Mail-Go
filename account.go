package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/logrusorgru/aurora"
	"log"
	"net/http"
	"strconv"
)

func Account(w http.ResponseWriter, r *http.Request) {
	var is string
	// Check if we should use `=` for a Wii or
	// `:` for the Homebrew patcher.
	if r.URL.Path == "/cgi-bin/account.cgi" {
		is = "="
	} else {
		is = ":"
	}

	wiiID := r.Form.Get("mlid")
	if !friendCodeIsValid(wiiID) {
		fmt.Fprint(w, GenAccountErrorCode(610, is, "Invalid Wii Friend Code."))
		return
	}

	w.Header().Add("Content-Type", "text/plain;charset=utf-8")

	stmt, err := db.Prepare("INSERT IGNORE INTO `accounts` (`mlid`,`passwd`, `mlchkid` ) VALUES (?, ?, ?)")
	if err != nil {
		fmt.Fprint(w, GenAccountErrorCode(410, is, "Database error."))
		log.Fatal(err)
		return
	}

	passwd := RandStringBytesMaskImprSrc(16)
	passwdByte := sha512.Sum512(append(salt, []byte(passwd)...))
	passwdHash := hex.EncodeToString(passwdByte[:])

	mlchkid := RandStringBytesMaskImprSrc(32)
	mlchkidByte := sha512.Sum512(append(salt, []byte(mlchkid)...))
	mlchkidHash := hex.EncodeToString(mlchkidByte[:])

	result, err := stmt.Exec(wiiID, passwdHash, mlchkidHash)
	if err != nil {
		fmt.Fprint(w, GenAccountErrorCode(410, is, "Database error."))
		log.Println(err)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		fmt.Fprint(w, GenAccountErrorCode(410, is, "Database error."))
		log.Println(err)
		return
	}

	if affected == 0 {
		fmt.Fprint(w, "\n",
			GenAccountErrorCode(211, is, "Duplicate registration."))
		return
	}

	fmt.Fprint(w, "\n",
		"cd", is, "100", "\n",
		"msg", is, "Success", "\n",
		"mlid", is, wiiID, "\n",
		"passwd", is, passwd, "\n",
		"mlchkid", is, mlchkid, "\n")
}

func GenAccountErrorCode(error int, is string, reason string) string {
	if error == 410 {
		log.Println(aurora.Red("[Warning]"), "Encountered error", error, "with reason", reason)
	}
	return fmt.Sprint(
		"cd", is, strconv.Itoa(error), "\n",
		"msg", is, reason, "\n")
}
