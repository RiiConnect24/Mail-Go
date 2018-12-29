package utilities

import (
	"fmt"
	"github.com/getsentry/raven-go"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"math/rand"
	"regexp"
	"runtime"
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

// friendCodeIsValid determines if a friend code is valid by
// checking not empty, is 17 in length, starts with w.
// BUG(spotlightishere): does not actually determine at a numerical level if valid.
func FriendCodeIsValid(wiiID string) bool {
	return mailRegex.MatchString(wiiID)
}

// random returns a random number in a range between two given integers.
func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

// GenerateBoundary returns a string with the format Nintendo used for boundaries.
func GenerateBoundary() string {
	return fmt.Sprint(time.Now().Format("200601021504"), "/", random(1000000, 9999999))
}

func LogError(ravenClient *raven.Client, reason string, err error) {
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