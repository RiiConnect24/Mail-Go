package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-systemd/daemon"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"os"
)

// Config structure for `config.json`.
type Config struct {
	Port           int
	Host           string
	Username       string
	Password       string
	DBName         string
	Interval       int
	BindTo         string
	SendGridKey    string
	SendGridDomain string
	Debug          bool
}

var global Config
var db *sql.DB

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if global.Debug {
			// Dump path, etc
			log.Printf("%s %s", r.Method, r.URL)
		}
		handler.ServeHTTP(w, r)
	})
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	Check(w, r, db, global.Interval)
}

func receiveHandler(w http.ResponseWriter, r *http.Request) {
	Receive(w, r, db)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	Delete(w, r, db)
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	Send(w, r, db, global)
}

func accountHandler(w http.ResponseWriter, r *http.Request) {
	Account(w, r, db)
}

func main() {
	file, err := os.Open("config/config.json")
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&global)
	if err != nil {
		panic(err)
	}
	testDb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		global.Username, global.Password, global.Host, global.Port, global.DBName))
	if err != nil {
		panic(err)
	}
	err = testDb.Ping()
	if err != nil {
		panic(err)
	}

	// If we've reached here, we're working fine.
	db = testDb

	log.Println("Running...")
	http.HandleFunc("/cgi-bin/account.cgi", accountHandler)
	http.HandleFunc("/cgi-bin/check.cgi", checkHandler)
	http.HandleFunc("/cgi-bin/receive.cgi", receiveHandler)
	http.HandleFunc("/cgi-bin/delete.cgi", deleteHandler)
	http.HandleFunc("/cgi-bin/send.cgi", sendHandler)

	// Allow systemd to run as notify
	// Thanks to https://vincent.bernat.im/en/blog/2017-systemd-golang
	// for the following things.
	daemon.SdNotify(false, "READY=1")

	// We do this to log all access to the page.
	log.Fatal(http.ListenAndServe(global.BindTo, logRequest(http.DefaultServeMux)))
}
