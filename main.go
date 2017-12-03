package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"os"
	"github.com/coreos/go-systemd/daemon"
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
}

var db *sql.DB
var global Config

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL)
		// TODO: remove header dumping
		for name, test := range r.Header {
			log.Printf("%s => %s", name, test)
		}
		handler.ServeHTTP(w, r)
	})
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	Check(w, r, global.Interval)
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
	file, err := os.Open("config.json")
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
	daemon.SdNotify(false, "READY=1")
	// We do this to log all access to the page.
	log.Fatal(http.ListenAndServe(global.BindTo, logRequest(http.DefaultServeMux)))
}
