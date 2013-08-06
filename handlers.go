//HTTP handlers

package main

import (
	"github.com/golang/glog"
	"fmt"
	"net/http"
	"strings"
	"time"
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

// If the URL is empty (just "/"), then it serves the index template
// with the movie names from the movies table. Otherwise, it serves
// the file named by the path
func mainHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAccess(w, r) {
		return
	}

	if len(r.URL.Path) == 1 {
		httpError := func(err error) {
			glog.Errorf("Error in main handler: %s", err)
			http.Error(w, fmt.Sprint("Failed to fetch movie names"), http.StatusInternalServerError)
		}
		// Reads the movies table to get all the movie names
		rows, err := selectStatements["getNames"].Query()
		if err != nil {
			httpError(err)
			return
		}
		movieNames := make([]string, 0)
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				httpError(err)
				return
			}
			if err = rows.Err(); err != nil {
				httpError(err)
				return
			}
			movieNames = append(movieNames, name)
		}
		if err = runTemplate("index", w, movieNames); err != nil {
			httpError(err)
		}
	} else {
		http.ServeFile(w, r, *srcPath + "/" + r.URL.Path[1:])
	}
}

// Serves the specified file, incrementing the download count.
// *moviePath should not be in the url
func fetchHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAccess(w, r) {
		return
	}

	filename := r.URL.Path[len(fetchPath):]
	filelocation := *moviePath + "/" + filename
	glog.V(infolevel).Infof("Fetching file: %s", filelocation)
	f, err := os.Open(filelocation)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not serve file %s", filename), http.StatusNotFound)
		return
	}
	defer f.Close()

	rs := io.ReadSeeker(f)

	w.Header().Set("Content-Type", "binary/octet-stream")
	http.ServeContent(w, r, filename, time.Time{}, rs)
	glog.V(infolevel).Infof("Served file: %s", filelocation)

	// Updates the download count, if no rows were affected, it
	// should have thrown the "could not serve file" error, so it
	// panics here
	res, err := insertStatements["addDownload"].Exec(filename)
	if err != nil {
		glog.Errorf("Error updating download count for %s: %s", filename, err)
	}
	rowcount, err := res.RowsAffected();
	if err != nil {
		glog.Error("Error retrieving rows affected for addDownload query")
	}
	if rowcount == 0 {
		panic("Update changed 0 rows, so it should have thrown an error above")
	}
}
