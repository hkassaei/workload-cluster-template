package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	_ "github.com/jackc/pgx/v4/stdlib"
)

// global db connection pool (using a global variable since it's a simple PoC)
var db *sql.DB

type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

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

func getAllAlbumsHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling get all albums at %s\n", req.URL.Path)
	allAlbums, _ := getAllAlbums()
	renderJSON(w, allAlbums)

}

func getAllAlbums() ([]Album, error) {

	// using the global connection pool 'db'
	rows, err := db.Query("SELECT * FROM album")
	if rows == nil {
		log.Print("zero rows read")
	}
	if err != nil {
		//fmt.Errorf("DB.Query: %v", err)
		return nil, err
	}
	defer rows.Close()
	var albums []Album
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, err
		}
		albums = append(albums, alb)

	}
	return albums, nil
}

func getVersion(w http.ResponseWriter, req *http.Request) {
	log.Printf("Getting version\n")
	version := "0.2.2"
	renderJSON(w, version)
}

func main() {
	var (
		dbUser    = os.Getenv("DB_USER")
		dbPwd     = os.Getenv("DB_PASS")
		dbTCPHost = "127.0.0.1" // since using cloudsql proxy
		dbPort    = "5432"
		dbName    = "demo_app_db"
	)

	dbURI := fmt.Sprintf("host=%s user=%s password=%s port=%s database=%s sslmode=disable",
		dbTCPHost, dbUser, dbPwd, dbPort, dbName)

	// initialize connection pool
	err := initDB(dbURI)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.HandleFunc("/albums", getAllAlbumsHandler).Methods("GET")
	router.HandleFunc("/version", getVersion).Methods("GET")

	log.Fatal(http.ListenAndServe(":8081", router))
}
