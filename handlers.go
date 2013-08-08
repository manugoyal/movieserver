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
	"time"
	"os"
	"io"
)

const (
	imagePath = "images/"
	loginPath = "/"
	mainPath = "/main/"
	fetchPath = "/fetch/"
	checkAccessPath = "/checkAccess/"
)

// Launches the login template when the user opens up http://[ip]:[port]/
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if err := runTemplate("login", w, nil); err != nil {
		return
	} 
}

// Makes sure client has valid username and password submitted
// on the login page. If not, an error message will be returned.
func checkAccessHandler(w http.ResponseWriter, r *http.Request) {	
	var user string
	user = r.FormValue("username")
	var password string
	password = r.FormValue("password")
	fmt.Println(password)
	row := selectStatements["getUserAndPassword"].QueryRow(user, password)
	var throwaway string
	if err := row.Scan(&throwaway); err != nil {
		glog.Error(err)
		fmt.Println(throwaway)
		http.Error(w, "Invalid username or passoword", http.StatusServiceUnavailable)
		return 
	}
	
	http.Redirect(w, r, mainPath, http.StatusFound)
}
type movieRow struct {
	Name string
	Downloads uint64
	}
// If the URL is empty (just "/"), then it serves the index template
// with the movie names from the movies table. Otherwise, it serves
// the file named by the path
func mainHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) == len(mainPath) {
		httpError := func(err error) {
			glog.Errorf("Error in main handler: %s", err)
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
		if err = runTemplate("index", w, movies); err != nil {
			httpError(err)
		}
	} else {
		http.ServeFile(w, r, *srcPath + "/" + r.URL.Path[1:])
	}
}

// Serves the specified file, incrementing the download count.
// *moviePath should not be in the url
func fetchHandler(w http.ResponseWriter, r *http.Request) {

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
