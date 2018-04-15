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
	"github.com/logrusorgru/aurora"
	"io/ioutil"
	"html/template"
	"github.com/RiiConnect24/Mail-Go/patch"
)

var global patch.Config
var db *sql.DB
var templates *template.Template

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse form for further usage.
		r.ParseForm()

		if global.Debug {
			log.Printf("%s %s", aurora.Blue(r.Method), aurora.Red(r.URL))
			for name, value := range r.Form {
				log.Print(name, " ", aurora.Green("=>"), " ", value)
			}
			log.Printf("Accessing from: %s", aurora.Blue(r.Host))
		}

		// Finally, serve.
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

func configHandle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		// todo: a u t h e n t i c a t i o n
		r.ParseForm()

		fileWriter, _, err := r.FormFile("uploaded_config")
		if err != nil {
			log.Printf("incorrect file: %v", err)
		}

		file, err := ioutil.ReadAll(fileWriter)
		if err != nil {
			log.Printf("unable to read file entirely: %v", err)
		}
		patched, err := patch.ModifyNwcConfig(file, db, global)
		if err != nil {
			log.Printf("unable to patch: %v", err)
			w.Write([]byte("It seems your patching went awry. Email devs@disconnect24.xyz to see if you can repatch."))
		}
		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", "attachment; filename=\"nwc24msg.cfg\"")
		w.Write(patched)
		break
	case "GET":
		fmt.Fprint(w, "This page doesn't do anything by itself. Try going to the main site.")
	default:
		break
	}
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
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		global.Username, global.Password, global.Host, global.Port, global.DBName))
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	// Load templates for HTML serving later on
	templateLocation := "/templates"
	templateDir, err := os.Open(templateLocation)
	if err != nil {
		panic(err)
	}
	templateDirList, err := templateDir.Readdir(-1)
	if err != nil {
		panic(err)
	}
	var templatePaths []string
	for _, templateFile := range templateDirList {
		templatePaths = append(templatePaths, fmt.Sprint(templateLocation, "/", templateFile.Name()))
	}
	templates, err = template.ParseFiles(templatePaths...)

	// Mail calls
	http.HandleFunc("/cgi-bin/account.cgi", Account)
	http.HandleFunc("/cgi-bin/check.cgi", checkHandler)
	http.HandleFunc("/cgi-bin/receive.cgi", receiveHandler)
	http.HandleFunc("/cgi-bin/delete.cgi", deleteHandler)
	http.HandleFunc("/cgi-bin/send.cgi", sendHandler)

	// Site
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// We only want the primary page.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		s1 := templates.Lookup("header.tmpl")
		s1.ExecuteTemplate(w, "header", nil)
		fmt.Println()
		s2 := templates.Lookup("patch.tmpl")
		s2.ExecuteTemplate(w, "content", nil)
		fmt.Println()
		s3 := templates.Lookup("footer.tmpl")
		s3.ExecuteTemplate(w, "footer", nil)
		fmt.Println()
		s3.Execute(w, nil)
	})
	http.HandleFunc("/patch", configHandle)

	// Allow systemd to run as notify
	// Thanks to https://vincent.bernat.im/en/blog/2017-systemd-golang
	// for the following things.
	daemon.SdNotify(false, "READY=1")
	log.Println("Running...")

	// We do this to log all access to the page.
	log.Fatal(http.ListenAndServe(global.BindTo, logRequest(http.DefaultServeMux)))
}
