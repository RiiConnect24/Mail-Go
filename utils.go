package main

import (
	"fmt"
	"github.com/RiiConnect24/wiino/golang"
	"github.com/getsentry/raven-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/logrusorgru/aurora"
	"log"
	"math/rand"
	"regexp"
	"runtime"
	"strconv"
	"time"
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
	switch error {
	case 220:
		break
	default:
		log.Println(aurora.Red("[Warning]"), "Encountered error", error, "with reason", reason)
	}
	return fmt.Sprint(
		"cd=", strconv.Itoa(error), "\n",
		"msg=", reason, "\n")
}

// GenSuccessResponse returns a successful message, using = as the divider between characters.
func GenSuccessResponse() string {
	return GenSuccessResponseTyped("=")
}

// GenSuccessResponseTyped returns a successful message, using the specified character as a divider.
func GenSuccessResponseTyped(divider string) string {
	return fmt.Sprint(
		"cd", divider, "100\n",
		"msg", divider, "Success.\n")
}

// friendCodeIsValid determines if a friend code is valid by
// checking not empty, is 17 in length, and starts with w.
// It then checks the numerical validity of the friend code.
func friendCodeIsValid(friendCode string) bool {
	// An empty or invalid length mlid is automatically false.
	if friendCode == "" || len(friendCode) != 17 {
		return false
	}

	// Ensure the provided mlid is the correct format.
	if !mailRegex.MatchString(friendCode) {
		return false
	}

	// We verified previously that the last 16 characters are digits. This should not fail.
	// However, should it, we do not want to hint to the user any error occurred and return false.
	wiiId, err := strconv.Atoi(friendCode[1:])
	if err != nil {
		return false
	}

	return wiino.NWC24CheckUserID(uint64(wiiId)) == 0
}

// GenerateBoundary returns a string with the format Nintendo used for boundaries.
func GenerateBoundary() string {
	return fmt.Sprint(time.Now().Format("200601021504"), "/", random(1000000, 9999999))
}

func LogError(reason string, err error) {
	// Adapted from
	// https://stackoverflow.com/a/38551362
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		// Log to console
		log.Printf("%s: %v", reason, err)

		// and if it's available, Sentry.
		if ravenClient != nil {
			raven.CaptureError(err, map[string]string{"given_reason": reason})
		}
	}
}
