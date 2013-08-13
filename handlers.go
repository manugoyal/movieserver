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
	"time"
)

const (
	mainURL        = "/main/"
	movieTableURL  = mainURL + "table/movie"
	movieURL       = mainURL + "movie/"
	loginURL       = "/"
	checkAccessURL = "/checkAccess/"
)

var (
	// Since running COUNT(*) on a very large movie entry set can
	// take a while, we store a mapping from where clauses to
	// index size, so that we don't have to rerun the COUNT(*)
	// when we're on the same movie set on a where clause we've
	// already seen. Every time indexMovies run, it will clear
	// this map, since it's creating a new movie set. This is
	// okay, because this map is really only useful for very large
	// movie sets, and reindexing very large indexes takes a
	// while, so it would be cleared infrequently
	fileIndexCount = make(map[string]uint64)
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
		if err := runTemplate("index", w, nil); err != nil {
			glog.Error(err)
			http.Error(w, "Failed to fetch home page", http.StatusInternalServerError)
		}
	} else {
		http.ServeFile(w, r, filepath.Join(*srcPath, r.URL.Path[len(mainURL):]))
	}
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
func addQueryParams(r *http.Request) (map[string]paramPair, error) {
	paramMap := make(map[string]paramPair)
	queryParams := r.URL.Query()
	// We implement searching via LIKE. REGEXP is too slow, since
	// it can't use an index. Since the filter uses wildcard
	// syntax, * corresponds to % and ? corresponds to _. We also
	// treat it as a prefix search, so we append a % to the string
	// always
	if filterString := queryParams.Get("q"); filterString != "" {
		fixedString := string(convertFilterString([]byte(filterString))) + "%"
		paramMap["where"] = paramPair{" AND path = ? AND name LIKE ?",
			[]interface{}{*moviePath, fixedString}}
	} else {
		paramMap["where"] = paramPair{}
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

// Serves the movies and downloads that are present from the movie
// table as a JSON object. It returns pagination settings for the
// client side paginator in the JSON object as well.
func movieTableHandler(w http.ResponseWriter, r *http.Request) {
	// No committing any reindexes from the heartbeat in between
	// queries here
	heartbeatLocks.fileIndexLock.Lock()
	defer heartbeatLocks.fileIndexLock.Unlock()

	httpError := func(err error) {
		glog.Errorf("Error in movieTable handler: %s", err)
		http.Error(w, fmt.Sprint("Failed to fetch movie names"), http.StatusInternalServerError)
	}

	// Get any additional query params as a query string
	paramMap, err := addQueryParams(r)
	if err != nil {
		httpError(err)
		return
	}

	// We need to first get the number of entries the query will
	// return. Sometimes, if the number of file entries decreased
	// since the client accessed a page, they could be accessing
	// an invalid page, so if that's the case, we change the page
	// in paginationState to 1 and use a limit offset of 0
	var total_entries uint64

	// First we check if the count is already in the
	// fileIndexCount map
	var fileIndexKey string
	if pp := paramMap["where"]; len(pp.args) == 2 {
		fileIndexKey = pp.args[0].(string) + pp.args[1].(string)
	}

	if count, ok := fileIndexCount[fileIndexKey]; ok {
		total_entries = count
	} else {
		// We need to run a COUNT(*) query. We only need the
		// WHERE param arg
		glog.V(vvLevel).Info("Running COUNT(*) over the movie index")
		countRow := dbHandle.QueryRow(fmt.Sprintf(sqlStatements["getMovieNum"], paramMap["where"].str),
			paramMap["where"].args...)
		if err := countRow.Scan(&total_entries); err != nil {
			httpError(err)
			return
		}
		// Update fileIndexCount
		fileIndexCount[fileIndexKey] = total_entries
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

	rows, err := dbHandle.Query(
		fmt.Sprintf(sqlStatements["getMovies"], paramMap["where"].str, paramMap["order"].str, paramMap["limit"].str),
		append(paramMap["where"].args, append(paramMap["order"].args, paramMap["limit"].args...)...)...)
	if err != nil {
		httpError(err)
		return
	}

	movies := make([]interface{}, 0)
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

	// Marshalls the json response, which is an array describing
	// the new pagination state plus the movies
	jsonResponse := []interface{}{paginationState, movies}
	jsonData, err := json.Marshal(jsonResponse)
	if err != nil {
		httpError(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}

// Serves the movie identified by the given pathname, incrementing the
// download count. *moviePath should not be in the url. If it's a
// directory, create a tar, skipping all the dotfiles and return that
func movieHandler(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Clean(r.URL.Path[len(movieURL):])
	filelocation := filepath.Join(*moviePath, filename)
	glog.V(vLevel).Infof("Fetching file: %s", filelocation)

	httpError := func(err error, code int) {
		glog.Errorf("Error in movie handler: %s", err)
		http.Error(w, fmt.Sprintf("Could not serve file %s", filename), code)
	}
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
		// When the user uncompresses the tar, it should
		// create the directory they selected to download, so
		// we split off the last part of the directory to use
		// as dirName in compressDir. If they're downloading
		// the root directory, we split one off the *moviePath
		var compressRoot, dirName string
		if filename == "." {
			compressRoot, dirName = filepath.Split(*moviePath)
		} else {
			compressRoot, dirName = filepath.Split(filename)
			compressRoot = filepath.Join(*moviePath, compressRoot)
		}
		if err := compressDir(compressRoot, dirName, tw); err != nil {
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
	res, err := dbHandle.Exec(sqlStatements["addDownload"], *moviePath, filename)
	if err != nil {
		glog.Errorf("Error updating download count for %s: %s", filename, err)
	}
	rowcount, err := res.RowsAffected()
	if err != nil {
		glog.Error("Error retrieving rows affected for addDownload query")
	}
	if rowcount == 0 {
		panic("Update changed 0 rows, so it should have thrown an error above")
	}
}

func setupHandlers() error {
	http.HandleFunc(mainURL, mainHandler)
	http.HandleFunc(movieTableURL, movieTableHandler)
	http.HandleFunc(movieURL, movieHandler)
	http.HandleFunc(loginURL, loginHandler)
	http.HandleFunc(checkAccessURL, checkAccessHandler)
	return nil
}
