package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"regexp"

	"github.com/RiiConnect24/Mail-Go/utilities"
)

var sendAuthRegex = regexp.MustCompile(`^mlid=(w\d{16})\r\npasswd=(.{16,32})$`)

func AuthForSend(mlid string) (bool, error) {
	// First, check if it's the send format of mlid.
	sendFormat := sendAuthRegex.FindStringSubmatch(mlid)
	if sendFormat != nil {
		// Format:
		// [0] = raw string
		// [1] = mlid match
		// [2] = passwd match
		mlid = sendFormat[1]
		passwd := sendFormat[2]
		return Auth(mlid, passwd)
	} else {
		// It's not send nor anything else we know at this point.
		return false, errors.New("invalid mail ID")
	}
}

// Auth is a function designed to parse potential information from
// a WC24 request, such as mlchkid and passwd.
// It takes a given type and attempts to correspond that to one recorded in a database.
func Auth(mlid string, passwd string) (bool, error) {
	if utilities.FriendCodeIsValid(mlid) {
		// Now we need to double check passwd also exists.
		if passwd == "" {
			return false, errors.New("invalid authentication type")
		}
	} else {
		// It's not send nor anything else we know at this point.
		return false, errors.New("invalid mail ID")
	}

	// If we're using passwd, we want to select passwd and mlid for security.
	// Grab salt + passwd sha512
	hashByte := sha512.Sum512(append(salt, []byte(passwd)...))
	hash := hex.EncodeToString(hashByte[:])

	stmt, err := db.Prepare("SELECT `passwd` FROM `accounts` WHERE `mlid` = ? AND `passwd` = ?")
	if err != nil {
		return false, err
	}

	var passwdResult string
	err = stmt.QueryRow(mlid, hash).Scan(&passwdResult)

	if err == sql.ErrNoRows {
		// Not found.
		return false, nil
	} else if err != nil {
		// Some type of SQL error... pass it on.
		return false, err
	} else {
		return true, nil
	}
}