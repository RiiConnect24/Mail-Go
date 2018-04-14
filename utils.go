package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
	_ "github.com/go-sql-driver/mysql"
	"regexp"
	"github.com/logrusorgru/aurora"
)

// https://stackoverflow.com/a/31832326/3874884
var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var mailRegex = regexp.MustCompile(`w\d{16}`)

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
		log.Println(aurora.Red("[Warning]"), "Encountered error", error, "with reason", reason)
	}

	return fmt.Sprint(
		"cd", mailNumber[1:], "=", strconv.Itoa(error), "\n",
		"msg", mailNumber[1:], "=", reason, "\n")
}

// GenNormalErrorCode formulates a proper response for overall errors.
func GenNormalErrorCode(error int, reason string) string {
	if error != 100 {
		log.Println(aurora.Red("[Warning]"), "Encountered error", error, "with reason", reason)
	}
	return fmt.Sprint(
		"cd=", strconv.Itoa(error), "\n",
		"msg=", reason, "\n")
}

// friendCodeIsValid determines if a friend code is valid by
// checking not empty, is 17 in length, starts with w.
// BUG(spotlightishere): does not actually determine at a numerical level if valid.
func friendCodeIsValid(wiiID string) bool {
	return mailRegex.MatchString(wiiID)
}
