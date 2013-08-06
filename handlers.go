//HTTP handlers

package main

import (
	"log"
	"fmt"
	"net/http"
	"strings"
	"time"
	"path/filepath"
	"os"
	"io"
)

const (
	imagePath = "images/"
	mainPath = "/"
	fetchPath = "/fetch/"
)

// Makes sure that the request's ip is allowed. Sends an error message
// if it isn't. Returns true if it is allowed, false if it isn't
func checkAccess(w http.ResponseWriter, r *http.Request) bool {
	ipstr := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
	if _, ok := allowedIP[ipstr]; !ok {
		http.Error(w, fmt.Sprint("You do not have access to this site"), http.StatusServiceUnavailable)
		return false
	}
	return true
}

// If the URL is empty (just "/"), then it reindexes the movie
// directory and serves index template with the movie names.
// Otherwise, it serves the file named by the path
func mainHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAccess(w, r) {
		return
	}
	if len(r.URL.Path) == 1 {
		log.Print("Indexing movie directory")
		movieNames = make([]string, 0)
		err := filepath.Walk(*moviePath, movieWalkFn)
		if err != nil {
			errmsg := "Failed to index movie directory"
			log.Printf(errmsg)
			http.Error(w, errmsg, http.StatusInternalServerError)
			return
		}
		runTemplate("index", w, movieNames)
	} else {
		http.ServeFile(w, r, *srcPath + "/" + r.URL.Path[1:])
	}
}

// Serves the specified file. *moviePath should not be in the url
func fetchHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAccess(w, r) {
		return
	}

	filename := r.URL.Path[len(fetchPath):]
	filelocation := *moviePath + "/" + filename
	log.Printf("Fetching file: %s", filelocation)
	f, err := os.Open(filelocation)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not serve file %s", filename), http.StatusNotFound)
		return
	}
	defer f.Close()

	rs := io.ReadSeeker(f)
	w.Header().Set("Content-Type", "binary/octet-stream")
	http.ServeContent(w, r, filename, time.Time{}, rs)
	log.Printf("Served file: %s", filelocation)
}
