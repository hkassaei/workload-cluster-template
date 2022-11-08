package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	// For a full list of sql database drivers in Go, refer to: https://github.com/golang/go/wiki/SQLDrivers
	_ "github.com/jackc/pgx/v4/stdlib"
)

// global db connection pool (using a global variable since it's a simple PoC)
var db *sql.DB

func initDB(dbConnection string) error {

	var err error
	// [START cloud_sql_postgres_databasesql_connect_tcp]
	// using the global db pool
	db, err = sql.Open("pgx", dbConnection)
	if err != nil {
		return err
	}
	// open does not actually try connecting to the database, call Ping to verify
	// intialization has been indeed successful
	return db.Ping()
}

func createDatabase(dbName string) error {
	if _, err := db.Exec("create database " + dbName); err != nil {
		return err
	}
	return nil
}

func createTable() error {
	if _, err := db.Exec(`create table shortened_urls (	id text primary key, url text not null)`); err != nil {
		return err
	}
	return nil
}

// renderJSON renders 'v' as JSON and writes it as a response into w.
func renderJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func getVersion(w http.ResponseWriter, req *http.Request) {
	log.Printf("Getting version\n")
	version := "0.2.4"
	renderJSON(w, version)
}

func getUrlHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var url string
	err := db.QueryRow("select url from shortened_urls where id=$1", vars["url"]).Scan(&url)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, url, http.StatusSeeOther)
}

func putUrlHandler(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["url"]
	var url string
	if body, err := io.ReadAll(req.Body); err == nil {
		url = string(body)
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := db.Exec(`insert into shortened_urls(id, url) values ($1, $2)
	on conflict (id) do update set url =excluded.url`, id, url); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func deleteUrlHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	if _, err := db.Exec("delete from shortened_urls where id=$1", vars["url"]); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func main() {
	var (
		dbUser    = os.Getenv("DB_USER")
		dbPwd     = os.Getenv("DB_PASS")
		dbTCPHost = "127.0.0.1" // since using cloudsql proxy
		dbPort    = "5432"
		dbName    = "demo_app_db"
	)

	dbURI := fmt.Sprintf("host=%s user=%s password=%s port=%s sslmode=disable",
		dbTCPHost, dbUser, dbPwd, dbPort)
	log.Println("database url:" + dbURI)

	// initialize connection pool without the database name
	if err := initDB(dbURI); err != nil {
		log.Fatal(err)
	} else if err := createDatabase(dbName); err != nil {
		log.Fatal(err)
	}
	db.Close()

	dbURI = fmt.Sprintf("host=%s user=%s password=%s port=%s database=%s sslmode=disable",
		dbTCPHost, dbUser, dbPwd, dbPort, dbName)
	log.Println("database url:" + dbURI)

	// initialize connection pool, this time with the database name
	if err := initDB(dbURI); err != nil {
		log.Fatal(err)
	} else if err := createTable(); err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.HandleFunc("/version", getVersion).Methods("GET")
	router.HandleFunc("/{url}", getUrlHandler).Methods("GET")
	router.HandleFunc("/{url}", putUrlHandler).Methods("PUT")
	router.HandleFunc("/{url}", deleteUrlHandler).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8080", router))
}
