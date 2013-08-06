// A web server that lets clients download movies
package main

import (
	"log"
	"net/http"
	"path/filepath"
	"flag"
)

const (
	port = ":8080"
)

var (
	srcPath = flag.String("src-path", srcdir(), "The path of the movieserver source directory")
	moviePath = flag.String("movie-path", "movies", "The path of the movies directory")
	movieNames []string
	allowedIP = map[string]bool{"[::1]": true, "98.236.150.191": true, "174.51.196.185": true}
)

func main() {
	flag.Parse()
	*srcPath = filepath.Clean(*srcPath)
	*moviePath = filepath.Clean(*moviePath)

	log.Print("Fetching html templates")
	err := fetchTemplates("index")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc(mainPath, mainHandler)
	http.HandleFunc(fetchPath, fetchHandler)

	log.Printf("Listening on port %s\n", port[1:])
	http.ListenAndServe(port, nil)
}
