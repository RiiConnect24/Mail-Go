package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"regexp"
)

func initAuthDB() {
	var err error
	validatePasswdStmt, err = db.Prepare("SELECT IF(EXISTS(SELECT passwd FROM accounts WHERE mlid = ? AND passwd = ?), 1, 0)")
	if err != nil {
		LogError("Unable to prepare auth statement", err)
		panic(err)
	}
}

var (
	validatePasswdStmt    *sql.Stmt
	ErrInvalidCredentials = errors.New("an authentication error occurred")
)

// sendAuthRegex describes a regex to validate a given mlid and passwd from the client.
// This technically should be mlid=w1234123412341234\r\npasswd=xyz, but \n is used
// for ease of interoperability with UNIX-centric clients.
var sendAuthRegex = regexp.MustCompile(`^mlid=(w\d{16})\r?\npasswd=(.{16,32})$`)

// parseSendAuth obtains a mlid and passwd from the given format.
// If it is unable to do so, it returns empty strings for both.
// It additionally determines whether the given mlid is valid -
// if not, it returns empty strings for both values as well.
func parseSendAuth(format string) (string, string) {
	match := sendAuthRegex.FindStringSubmatch(format)
	if match != nil {
		// Format:
		// [0] = raw string
		// [1] = mlid match
		// [2] = passwd match
		return match[1], match[2]
	} else {
		return "", ""
	}
}

// hashAuthParam salts and hashes the passed parameter appropriately.
func hashAuthParam(param string) string {
	hashByte := sha512.Sum512(append(salt, []byte(param)...))
	return hex.EncodeToString(hashByte[:])
}

// checkPasswdValidity returns an error if credentials are invalid,
// or a database error occurred. If not, it returns nil.
func checkPasswdValidity(mlid string, passwd string) error {
	if mlid == "" || passwd == "" || !friendCodeIsValid(mlid) {
		return ErrInvalidCredentials
	}

	passwdHash := hashAuthParam(passwd)

	// Query the database.
	exists := false
	result := validatePasswdStmt.QueryRow(mlid, passwdHash)
	err := result.Scan(&exists)
	if err != nil {
		return err
	}

	// Return our queried result.
	if exists {
		return nil
	} else {
		return ErrInvalidCredentials
	}
}
