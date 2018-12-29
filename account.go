package main

import (
	"crypto/sha512"
	"encoding/hex"
	"github.com/RiiConnect24/Mail-Go/utilities"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
)

func Account(c *gin.Context) {
	var is string
	// Check if we should use `=` for a Wii or
	// `:` for the Homebrew patcher.
	if c.Request.URL.Path == "/cgi-bin/account.cgi" {
		is = "="
	} else {
		is = ":"
	}

	wiiID := c.PostForm("mlid")
	if !utilities.FriendCodeIsValid(wiiID) {
		TypedErrorResponse(c, 610, is, "Invalid Wii Friend Code.")
		return
	}

	c.Header("Content-Type", "text/plain;charset=utf-8")

	stmt, err := db.Prepare("INSERT IGNORE INTO `accounts` (`mlid`,`passwd`, `mlchkid` ) VALUES (?, ?, ?)")
	if err != nil {
		TypedErrorResponse(c, 410, is, "Database error.")
		utilities.LogError(ravenClient, "Unable to prepare account statement", err)
		return
	}

	passwd := utilities.RandStringBytesMaskImprSrc(16)
	passwdByte := sha512.Sum512(append(salt, []byte(passwd)...))
	passwdHash := hex.EncodeToString(passwdByte[:])

	mlchkid := utilities.RandStringBytesMaskImprSrc(32)
	mlchkidByte := sha512.Sum512(append(salt, []byte(mlchkid)...))
	mlchkidHash := hex.EncodeToString(mlchkidByte[:])

	result, err := stmt.Exec(wiiID, passwdHash, mlchkidHash)
	if err != nil {
		TypedErrorResponse(c, 410, is, "Database error.")
		utilities.LogError(ravenClient, "Unable to execute statement", err)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		TypedErrorResponse(c, 410, is, "Database error.")
		utilities.LogError(ravenClient, "Unable to get rows affected", err)
		return
	}

	if affected == 0 {
		TypedErrorResponse(c, 211, is, "Duplicate registration.")
		return
	}

	c.String(http.StatusOK, fmt.Sprint("cd", is, "100", "\n",
		"msg", is, "Success.", "\n",
		"mlid", is, wiiID, "\n",
		"passwd", is, passwd, "\n",
		"mlchkid", is, mlchkid, "\n"))
}
