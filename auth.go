package main

import (
	"golang.org/x/crypto/bcrypt"
	"errors"
	"log"
	"net/url"
	"database/sql"
)

// Auth is a function designed to parse potential information from
// a WC24 request, such as mlchkid and passwd.
// It takes a given type and attempts to correspond that to one recorded in a database.
func Auth(form url.Values) (bool, error) {
	// Figure out what part of authorization we're supposed to figure out.
	log.Println("Using type passwd")

	mlid := form.Get("mlid")
	if !friendCodeIsValid(mlid) {
		return false, errors.New("invalid mail ID")
	}

	// Now we need to double check the given auth type is even used.
	formGivenType := form.Get("passwd")
	if formGivenType == "" {
		return false, errors.New("invalid authentication type")
	}

	// If we're using passwd, we want to select passwd and mlid for security.
	// Since we only have mlkchkid for check, it's the best we can do.
	stmt, err := db.Prepare("SELECT `passwd` FROM `accounts` WHERE `mlid` = ?")
	if err != nil {
		return false, err
	}

	var passwdResult []byte
	err = stmt.QueryRow(mlid).Scan(&passwdResult)

	if err == sql.ErrNoRows {
		// Not found.
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		// Found.
		// We now need to double check what was given.
		return bcrypt.CompareHashAndPassword(passwdResult, []byte(formGivenType)) != nil, nil
	}
}
