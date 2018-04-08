package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
	 "golang.org/x/crypto/bcrypt"
	"net/http"
)

// https://stackoverflow.com/a/31832326/3874884
var src = rand.NewSource(time.Now().UnixNano())

var db *sql.DB

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandStringBytesMaskImprSrc makes a random string with the specified size.
func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// GenMailErrorCode formulates a proper response needed for mail-specific errors.
func GenMailErrorCode(mailNumber string, error int, reason string) string {
	if error != 100 {
		log.Println("[Warning] Encountered error", error, "with reason", reason)
	}

	return fmt.Sprint(
		"cd", mailNumber[1:], "=", strconv.Itoa(error), "\n",
		"msg", mailNumber[1:], "=", reason, "\n")
}

// GenNormalErrorCode formulates a proper response for overall errors.
func GenNormalErrorCode(error int, reason string) string {
	if error != 100 {
		log.Println("[Warning] Encountered error", error, "with reason", reason)
	}
	return fmt.Sprint(
		"cd=", strconv.Itoa(error), "\n",
		"msg=", reason, "\n")
}

func AccInsert(r *http.Request) sql.Result {
	mlchkid, err := bcrypt.GenerateFromPassword([]byte(r.Form.Get("mlchkid")), 10)
	insertAcc, err := db.Exec("INSERT INTO `accounts` (`mlid`, `password`, `mlchkid`) VALUES (?, ?, ?)", r.Form.Get("mlid"), nil, mlchkid)
	if err != nil {
		log.Fatal(err)
	}
	return insertAcc
}

func AccQuery(r *http.Request) *sql.Row {
	stmt := db.QueryRow("SELECT * FROM `accounts` WHERE `mlid` =?", r.Form.Get("mlid"))
	return stmt
}

func AccUpdate(r *http.Request) int {
	passwd, err := bcrypt.GenerateFromPassword([]byte(r.Form.Get("passwd")),10)
	if err != nil {
		log.Fatal(err)
	}
	_, err2 := db.Exec("UPDATE `accounts` SET `password` = ? WHERE `mlid` = ?", passwd, r.Form.Get("mlid"))
	if err2 != nil {
		log.Fatal(err2)
	}
	return 0
}

func Auth(w http.ResponseWriter, r *http.Request, isCheck bool) {
	stmt := AccQuery(r)
	type Login struct {
		mlid, mlchkid, passwd []byte
	}
	var l Login
	err := stmt.Scan(&l.mlid, &l.mlchkid, &l.passwd)
	if err == sql.ErrNoRows {
		if isCheck {
			AccInsert(r)
			AccQuery(r)
			return
		}
	}
	if isCheck {
		if bcrypt.CompareHashAndPassword(l.mlchkid, []byte(r.Form.Get("mlchkid"))) != nil {
			w.Write([]byte("Couldn't authenticate you.")) // Change later
			return
		}
	} else {
		if l.passwd == nil {
			AccUpdate(r)
			return
		}
		if bcrypt.CompareHashAndPassword(l.passwd, []byte(r.Form.Get("passwd"))) != nil {
			w.Write([]byte("Couldn't authenticate you.")) // Change later
			return
		}
	}
}