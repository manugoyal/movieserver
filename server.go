package main

import (
        "fmt"
        "log"
        "net/http"
)

const (
        port = ":8080"
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, world")
}

func main() {
        http.HandleFunc("/", mainHandler)
        log.Printf("Listening on port %s\n", port[1:])
	http.ListenAndServe(port, nil)
}



















