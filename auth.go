package main

import (
	"errors"
	"net/url"
	"database/sql"
	"crypto/sha512"
	"encoding/hex"
)

// Auth is a function designed to parse potential information from
// a WC24 request, such as mlchkid and passwd.
// It takes a given type and attempts to correspond that to one recorded in a database.
func Auth(form url.Values) (bool, error) {
	mlid := form.Get("mlid")
	if !friendCodeIsValid(mlid) {
		return false, errors.New("invalid mail ID")
	}

	// Now we need to double check the given auth type is even used.
	passwd := form.Get("passwd")
	if passwd == "" {
		return false, errors.New("invalid authentication type")
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
