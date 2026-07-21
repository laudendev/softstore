package main


import (
   "fmt"
   "log"
   "net/http"

   "softstore/internal/db"
)

func main() {
    database, err := db.Open("softstore.db")
    if err != nil {
	 log.Fatal(err)
    }
    defer database.Close()

    mux := http.NewServeMux()

    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	    fmt.Fprintln(w, "ok")
    })

    log.Println("listening on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
	    log.Fatal(err)
    }
}
