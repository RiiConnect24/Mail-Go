package main

import (
	"net/http"
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	"errors"
	"strings"
)

const (
	TypePasswd  = 1
	TypeMlchkid = 2
)

// Auth is a function designed to parse potential information from
// a WC24 request, such as mlchkid and passwd.
// It takes a given type and attempts to correspond that to one recorded in a database.
func Auth(r *http.Request, authType int) (bool, error) {
	mlid := r.Form.Get("mlid")
	if !friendCodeIsValid(mlid) {
		return false, errors.New("invalid mail ID")
	}

	// Figure out what part of authorization we're supposed to figure out.
	var authTypeAsString string
	if authType == TypePasswd {
		authTypeAsString = "passwd"
	} else if authType == TypeMlchkid {
		authTypeAsString = "mlchkid"
	} else {
		return false, errors.New("this isn't even a valid auth type, what're you doing")
	}

	// Now we need to double check the given auth type is even used.
	formGivenType := r.Form.Get(authTypeAsString)
	if formGivenType == "" {
		return false, errors.New("invalid authentication type")
	}

	// We're using the IP to associate a mlchkid with a password.
	ip, err := getIPAddress(r)
	if err != nil {
		return false, err
	}

	stmt, err := db.Prepare("SELECT passwd, mlchkid FROM `accounts` WHERE `ip` = INET_ATON(?) AND `mlid` = ?")
	if err != nil {
		return false, err
	}

	var typeResult []byte
	results, err := stmt.Query(ip, authTypeAsString)
	if err == sql.ErrNoRows {
		// Well.. since no one else is currently inserting
		// authentication data, we're hacking together an
		// IP/mlid based hack.

		// Let's go ahead and insert that.
		stmtString := "INSERT INTO `accounts` (`ip`, `mlid`, `type`) VALUES (INET_ATON('?'), '?', '?') ON DUPLICATE KEY UPDATE type = VALUES(type)"
		typeStmtString := strings.Replace(stmtString, "type", authTypeAsString, -1)
		stmt, err := db.Prepare(typeStmtString)
		if err != nil {
			return false, err
		}
		_, err = stmt.Exec(ip, mlid, formGivenType)
		if err != nil {
			return false, err
		}

		typeResult = []byte(formGivenType)
	} else {
		// We assume that there's only one result.
		// If there's more, whoops.
		var mlchkid []byte
		var passwd []byte
		results.Scan(&passwd, &mlchkid)

		if authType == TypePasswd {
			typeResult = passwd
		} else {
			typeResult = mlchkid
		}
	}

	return bcrypt.CompareHashAndPassword(typeResult, []byte(formGivenType)) != nil, nil
}
