/*
Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
*/

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
	"encoding/json"
)

const (
	mainURL = "/"
	moviesTableURL = "/table/movies"
	movieURL = "/movie/"
)

// Makes sure that the request's ip is allowed. Sends an error message
// if it isn't. Returns an error if it isn't
func checkAccess(w http.ResponseWriter, r *http.Request) error {
	if *unblockIPs {
		return nil
	}
	ipstr := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
	row := selectStatements["getAddr"].QueryRow(ipstr)
	var throwaway string
	if err := row.Scan(&throwaway); err != nil {
		glog.Error(err)
		http.Error(w, "You do not have access to this site", http.StatusServiceUnavailable)
		return fmt.Errorf("IP %s was not found", ipstr)
	}
	return nil
}

type movieRow struct {
	Name string
	Downloads uint64
}

// If the URL is empty (just "/"), then it serves the index template.
// Otherwise, it serves the file named by the path.
func mainHandler(w http.ResponseWriter, r *http.Request) {
	if err := checkAccess(w, r); err != nil {
		glog.Error(err)
		return
	}
	if len(r.URL.Path) == len(mainURL) {
		if err := runTemplate("index", w, nil); err != nil {
			glog.Error(err)
			http.Error(w, fmt.Sprint("Failed to fetch home page"), http.StatusInternalServerError)
		}
	} else {
		http.ServeFile(w, r, *srcPath + "/" + r.URL.Path[len(mainURL):])
	}
}

// Serves the movies and downloads that are present from the movies
// table as a JSON object
func moviesTableHandler(w http.ResponseWriter, r *http.Request) {
	if err := checkAccess(w, r); err != nil {
		glog.Error(err)
		return
	}
	httpError := func(err error) {
		glog.Errorf("Error in moviesTable handler: %s", err)
		http.Error(w, fmt.Sprint("Failed to fetch movie names"), http.StatusInternalServerError)
	}

	// Reads the movies table to get all the movie names and downloads
	rows, err := selectStatements["getMovies"].Query()
	if err != nil {
		httpError(err)
		return
	}
	movies := make([]movieRow, 0)
	for rows.Next() {
		var r movieRow
		if err = rows.Scan(&r.Name, &r.Downloads); err != nil {
			httpError(err)
			return
		}
		movies = append(movies, r)
	}
	if err = rows.Err(); err != nil {
		httpError(err)
		return
	}

	// Marshalls the movies slice into an array of JSON objects
	jsonData, err := json.Marshal(movies)
	if err != nil {
		httpError(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}

// Serves the movie identified by the given pathname, incrementing the
// download count. *moviePath should not be in the url
func movieHandler(w http.ResponseWriter, r *http.Request) {
	if err := checkAccess(w, r); err != nil {
		glog.Error(err)
		return
	}

	filename := r.URL.Path[len(movieURL):]
	filelocation := *moviePath + "/" + filename
	glog.V(infoLevel).Infof("Fetching file: %s", filelocation)
	f, err := os.Open(filelocation)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not serve file %s", filename), http.StatusNotFound)
		return
	}
	defer f.Close()

	rs := io.ReadSeeker(f)

	w.Header().Set("Content-Type", "binary/octet-stream")
	http.ServeContent(w, r, filename, time.Time{}, rs)
	glog.V(infoLevel).Infof("Served file: %s", filelocation)

	// Updates the download count, if no rows were affected, it
	// should have thrown the "could not serve file" error, so it
	// panics here
	res, err := insertStatements["addDownload"].Exec(*moviePath, filename)
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

func installHandlers() {
	http.HandleFunc(mainURL, mainHandler)
	http.HandleFunc(moviesTableURL, moviesTableHandler)
	http.HandleFunc(movieURL, movieHandler)
}
