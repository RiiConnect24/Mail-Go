package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
)

// https://stackoverflow.com/a/31832326/3874884
var src = rand.NewSource(time.Now().UnixNano())

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

func Auth(w http.ResponseWriter, r *http.Request, mode int) int {
	var mlchkid []byte
	var passwd []byte

	err := db.QueryRow("SELECT passwd,mlchkid FROM `accounts` WHERE `mlid` = ?", r.Form.Get("mlid")).Scan(&passwd, &mlchkid)

	if err == sql.ErrNoRows || passwd == nil {
		Account(w, r, db, mode)
		return 1
	} else if err != nil {
		GenNormalErrorCode(410, "Database error.")
		log.Fatal(err)
	}

	if mode == 1 {
		if bcrypt.CompareHashAndPassword([]byte(mlchkid), []byte(r.Form.Get("mlchkid"))) != nil {
			return 2
		}
	} else if mode == 2 {
		if bcrypt.CompareHashAndPassword([]byte(passwd), []byte(r.Form.Get("passwd"))) != nil {
			return 3
		}
	}

	return 0
}