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
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	mainURL        = "/main/"
	tableURL       = mainURL + "table/"
	movieURL       = mainURL + "movie/"
	tableKeysURL   = mainURL + "tableKeys/"
	loginURL       = "/"
	checkAccessURL = "/checkAccess/"
)

// Launches the login template when the user opens up http://[ip]:[port]/
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if err := runTemplate("login", w, nil); err != nil {
		glog.Error(err)
		http.Error(w, "Failed to fetch login page", http.StatusInternalServerError)
	}
}

// Makes sure client has valid username and password submitted
// on the login page. If not, an error message will be returned.
func checkAccessHandler(w http.ResponseWriter, r *http.Request) {
	user, password := r.FormValue("username"), r.FormValue("password")
	row := dbHandle.QueryRow(sqlStatements["getUserAndPassword"], user, password)
	var throwaway string
	if err := row.Scan(&throwaway); err != nil {
		glog.Error(err)
		http.Error(w, "Invalid username or password", http.StatusForbidden)
		return
	}
	http.Redirect(w, r, mainURL, http.StatusFound)
}

type movieRow struct {
	Name      string `json:"name"`
	Downloads uint64 `json:"downloads"`
}

// If the URL is empty (just mainURL), then it serves the index
// template. Otherwise, it serves the file named by the path
func mainHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) == len(mainURL) {
		http.ServeFile(w, r, filepath.Join(*srcPath, "frontend", "templates", "index.html"))
	} else {
		http.ServeFile(w, r, filepath.Join(*srcPath, r.URL.Path[len(mainURL):]))
	}
}

// Returns a json array of the moviePaths keys
func tableKeysHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse := make([]interface{}, 0, len(moviePaths))
	for k, _ := range moviePaths {
		jsonResponse = append(jsonResponse, k)
	}

	jsonData, err := json.Marshal(jsonResponse)
	if err != nil {
		glog.Error(err)
		http.Error(w, "Failed to fetch table keys", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}

// Replaces * with % and ? with _, handling escaping correctly
func convertFilterString(filterString []byte) (result []byte) {
	result = make([]byte, len(filterString))
	for fInd := 0; fInd < len(filterString); fInd++ {
		switch filterString[fInd] {
		case '*':
			result[fInd] = '%'
		case '?':
			result[fInd] = '_'
		default:
			result[fInd] = filterString[fInd]
		}
		// If it's a backslash, we copy the next character verbatim
		if filterString[fInd] == '\\' && fInd+1 < len(filterString) {
			result[fInd+1] = filterString[fInd+1]
			fInd++
		}
	}
	return
}

type paramPair struct {
	str  string
	args []interface{}
}

// Looking at the form values in a request, it returns a map of SQL
// clauses to a pair of the string of the clause and its query params.
// So far it checks for q (a filter string), page and per_page
// (paging info), and sort_by and order (sorting)
func addQueryParams(r *http.Request, moviePath string) (map[string]paramPair, error) {
	paramMap := make(map[string]paramPair)
	queryParams := r.URL.Query()
	// We implement searching via LIKE. REGEXP is too slow, since
	// it can't use an index. Since the filter uses wildcard
	// syntax, * corresponds to % and ? corresponds to _. We also
	// treat it as a prefix search, so we append a % to the string
	// always
	if filterString := queryParams.Get("q"); filterString != "" {
		fixedString := string(convertFilterString([]byte(filterString))) + "%"
		paramMap["where"] = paramPair{" path = ? AND name LIKE ?",
			[]interface{}{moviePath, fixedString}}
	} else {
		paramMap["where"] = paramPair{" path = ?", []interface{}{moviePath}}
	}

	if sort_col, order := queryParams.Get("sort_by"), queryParams.Get("order"); len(sort_col+order) > 0 {
		paramMap["order"] = paramPair{str: fmt.Sprintf(" ORDER BY `%s` %s", sort_col, order)}
	} else {
		paramMap["order"] = paramPair{}
	}

	if page, per_page := queryParams.Get("page"), queryParams.Get("per_page"); len(page+per_page) > 0 {
		page_num, err := strconv.ParseUint(page, 10, 64)
		if err != nil {
			return nil, err
		}
		per_page_num, err := strconv.ParseUint(per_page, 10, 64)
		if err != nil {
			return nil, err
		}
		paramMap["limit"] = paramPair{" LIMIT ?, ?",
			[]interface{}{(page_num - 1) * per_page_num, per_page_num}}
	} else {
		paramMap["limit"] = paramPair{}
	}

	return paramMap, nil
}

// Serves the movies and downloads of the requested table from the
// movie table as a JSON object. It returns pagination settings for
// the client side paginator object in the JSON as well. The first
// segment in the url is the key of the movie path.
func tableHandler(w http.ResponseWriter, r *http.Request) {
	httpError := func(err error, code int) {
		glog.Errorf("Error in table handler: %s", err)
		http.Error(w, fmt.Sprint("Failed to fetch movie names"), code)
	}

	// It takes away any slashes from the key
	moviePathKey := strings.Replace(r.URL.Path[len(tableURL):], "/", "", -1)
	moviePath, ok := moviePaths[moviePathKey]
	if !ok {
		httpError(fmt.Errorf("Invalid key name: %s", moviePathKey), http.StatusBadRequest)
		return
	}
	// Get any additional query params as a query string
	paramMap, err := addQueryParams(r, moviePath)
	if err != nil {
		httpError(err, http.StatusInternalServerError)
		return
	}

	// Performs the select queries under a repeatable-read
	// transaction, so their results remain consistent
	trans, err := dbHandle.Begin()
	if err != nil {
		httpError(err, http.StatusInternalServerError)
		return
	}

	// We need to first get the number of entries the query will
	// return. Sometimes, if the number of file entries decreased
	// since the client accessed a page, they could be accessing
	// an invalid page, so if that's the case, we change the page
	// in paginationState to 1 and use a limit offset of 0
	var total_entries uint64

	countRow := trans.QueryRow(fmt.Sprintf(sqlStatements["getMovieNum"], paramMap["where"].str), paramMap["where"].args...)
	if err := countRow.Scan(&total_entries); err != nil {
		httpError(err, http.StatusInternalServerError)
		return
	}
	paginationState := map[string]interface{}{
		"total_entries": total_entries,
	}
	// If there's a limit clause that's out of bounds, make the
	// offset 0 and adjust the paginationState accordingly
	if pp := paramMap["limit"]; len(pp.args) == 2 {
		offset, limit := pp.args[0].(uint64), pp.args[1].(uint64)
		if offset >= total_entries {
			// We're out of bounds, change args[0]
			// (offset) to 0, and paginationState["page"]
			// to 1. We also need explicitly set per_page,
			// because otherwise backbone-paginator will
			// reset it incorrectly
			pp.args[0] = 0
			paginationState["page"] = 1
			paginationState["per_page"] = limit
		}
	}

	rows, err := trans.Query(
		fmt.Sprintf(sqlStatements["getMovies"], paramMap["where"].str, paramMap["order"].str, paramMap["limit"].str),
		append(paramMap["where"].args, append(paramMap["order"].args, paramMap["limit"].args...)...)...)
	if err != nil {
		httpError(err, http.StatusInternalServerError)
		return
	}

	movies := make([]interface{}, 0)
	for rows.Next() {
		var r movieRow
		if err = rows.Scan(&r.Name, &r.Downloads); err != nil {
			httpError(err, http.StatusInternalServerError)
			return
		}
		movies = append(movies, r)
	}
	if err = rows.Err(); err != nil {
		httpError(err, http.StatusInternalServerError)
		return
	}

	if err := trans.Commit(); err != nil {
		httpError(err, http.StatusInternalServerError)
		return
	}

	// Marshalls the json response, which is an array describing
	// the new pagination state plus the movies
	jsonResponse := []interface{}{paginationState, movies}
	jsonData, err := json.Marshal(jsonResponse)
	if err != nil {
		httpError(err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}

// Serves the movie identified by the given pathname, incrementing the
// download count. The keyname of the path should be be the first
// segment in the url, and the path of the file should be everything
// after that. If it's a directory, we create a tar, skipping all the
// dotfiles, and return that
func movieHandler(w http.ResponseWriter, r *http.Request) {
	httpError := func(err error, code int) {
		glog.Errorf("Error in movie handler: %s", err)
		http.Error(w, fmt.Sprintf("Could not serve request %s", r.URL.Path), code)
	}

	rest := r.URL.Path[len(movieURL):]
	var moviePathKey, filename string
	if slashIndex := strings.Index(rest, "/"); slashIndex == -1 {
		// The movie path key must be the last segment in the
		// URL, and there's no trailing slash. Assumes the
		// filename will be the directory named by moviePath
		// itself
		moviePathKey = rest
		filename = filepath.Clean("")
	} else {
		moviePathKey = rest[:slashIndex]
		filename = filepath.Clean(rest[len(moviePathKey)+1:])
	}
	moviePath, ok := moviePaths[moviePathKey]
	if !ok {
		httpError(fmt.Errorf("Could not find movie path key: %s", moviePathKey), http.StatusBadRequest)
		return
	}
	filelocation := filepath.Join(moviePath, filename)
	glog.V(vLevel).Infof("Fetching file: %s", filelocation)

	fi, err := os.Stat(filelocation)
	if err != nil {
		httpError(err, http.StatusNotFound)
		return
	}
	var (
		rs        io.ReadSeeker
		servename string
	)

	// If the named movie is a directory, it creates a tar out of
	// the directory and serves that. Otherwise it opens the file
	// and serves that.
	if fi.IsDir() {
		servename = filename + ".tar"
		// Creates the tar in a bytes.Buffer, then serves it
		// as a bytes.Reader
		buf := new(bytes.Buffer)
		tw := tar.NewWriter(buf)
		if err := tarDir(filepath.Join(moviePath, filename), tw); err != nil {
			httpError(err, http.StatusInternalServerError)
			return
		}
		tw.Close()
		rs = bytes.NewReader(buf.Bytes())
	} else {
		f, err := os.Open(filelocation)
		if err != nil {
			httpError(err, http.StatusNotFound)
			return
		}
		defer f.Close()
		rs, servename = f, filename
		// We want to serve the file in a way that will force
		// a download
		w.Header().Set("Content-Type", "binary/octet-stream")
	}

	http.ServeContent(w, r, servename, time.Time{}, rs)
	glog.V(vLevel).Infof("Served file: %s", filelocation)

	// Updates the download count, if no rows were affected, it
	// should have thrown the "could not serve file" error, so it
	// panics here
	res, err := dbHandle.Exec(sqlStatements["addDownload"], moviePath, filename)
	if err != nil {
		glog.Errorf("Error updating download count for %s: %s", filename, err)
		return;
	}
	rowcount, err := res.RowsAffected()
	if err != nil {
		glog.Error("Error retrieving rows affected for addDownload query")
		return;
	}
	if rowcount == 0 {
		panic("Update changed 0 rows, so it should have thrown an error above")
	}
}

func setupHandlers() error {
	http.HandleFunc(mainURL, mainHandler)
	http.HandleFunc(tableURL, tableHandler)
	http.HandleFunc(movieURL, movieHandler)
	http.HandleFunc(tableKeysURL, tableKeysHandler)
	http.HandleFunc(loginURL, loginHandler)
	http.HandleFunc(checkAccessURL, checkAccessHandler)
	return nil
}
