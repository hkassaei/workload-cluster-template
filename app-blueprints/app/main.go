package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/v1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "v0.1.1")
	})

	log.Fatal(http.ListenAndServe(":8081", nil))

}
