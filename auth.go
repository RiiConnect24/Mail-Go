package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
)

func initAuthDB() {
	var err error
	validatePasswdStmt, err = db.Prepare("SELECT `passwd` FROM `accounts` WHERE `mlid` = ? AND `passwd` = ?")
	if err != nil {
		LogError("Unable to prepare auth statement", err)
		panic(err)
	}
}

var validatePasswdStmt *sql.Stmt

var sendAuthRegex = regexp.MustCompile(`^mlid=(w\d{16})\r\npasswd=(.{16,32})$`)

// Auth is a function designed to parse potential information from
// a WC24 request, such as mlchkid and passwd.
// It takes a given type and attempts to correspond that to one recorded in a database.
// Returns whether or not auth was successful, if so the verified mlid, and any error.
func Auth(form url.Values) (bool, string, error) {
	mlid := form.Get("mlid")
	var passwd string

	// First, check if it's the send format of mlid.
	sendFormat := sendAuthRegex.FindStringSubmatch(mlid)
	if sendFormat != nil {
		// Format:
		// [0] = raw string
		// [1] = mlid match
		// [2] = passwd match
		mlid = sendFormat[1]
		passwd = sendFormat[2]
	} else if friendCodeIsValid(mlid) {
		// Now we need to double check passwd also exists.
		passwd = form.Get("passwd")
		if passwd == "" {
			return false, "", errors.New("invalid authentication type")
		}
	} else {
		// It's not send nor anything else we know at this point.
		return false, "", errors.New("invalid mail ID")
	}

	// If we're using passwd, we want to select passwd and mlid for security.
	// Grab salt + passwd sha512
	hashByte := sha512.Sum512(append(salt, []byte(passwd)...))
	hash := hex.EncodeToString(hashByte[:])

	var passwdResult string
	err := validatePasswdStmt.QueryRow(mlid, hash).Scan(&passwdResult)

	if err == sql.ErrNoRows {
		// Not found.
		return false, "", nil
	} else if err != nil {
		// Some type of SQL error... pass it on.
		return false, "", err
	} else {
		return true, mlid, nil
	}
}
