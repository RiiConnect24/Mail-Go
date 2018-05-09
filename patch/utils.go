package patch

import (
	"time"
	"math/rand"
	"runtime"
	"log"
	"github.com/getsentry/raven-go"
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

func LogError(ravenClient *raven.Client,reason string, err error) {
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