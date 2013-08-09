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
	"strconv"
)

const (
	mainURL = "/"
	moviesTableURL = "/table/movies"
	movieURL = "/movie/"
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

// Makes sure that the request's ip is allowed. Sends an error message
// if it isn't. Returns an error if it isn't
func checkAccess(w http.ResponseWriter, r *http.Request) error {
	if *unblockIPs {
		return nil
	}
	ipstr := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
	row := dbHandle.QueryRow(sqlStatements["getAddr"], ipstr)
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
		if filterString[fInd] == '/' && fInd+1 < len(filterString) {
			result[fInd+1] = filterString[fInd+1]
			fInd++
		}
	}
	return
}

type paramPair struct {
	paramString string
	paramArgs []interface{}
}
// Returns a map of clauses to a pair of the string of the clause and
// its query params
func addQueryParams(r *http.Request) (map[string]paramPair, error) {
	paramMap := make(map[string]paramPair)
	queryParams := r.URL.Query()
	fmt.Println(queryParams)
	// We implement searching via LIKE. REGEXP is too slow, since
	// it can't use an index. Since the filter uses wildcard
	// syntax, * corresponds to % and ? corresponds to _. We also
	// treat it as a prefix search, so we append a % to the string
	// always
	if filterString := queryParams.Get("q"); filterString != "" {
		fixedString := string(convertFilterString([]byte(filterString))) + "%"
		paramMap["where"] = paramPair{" AND path = ? AND name LIKE ?",
			[]interface{}{*moviePath, fixedString}}
	}

	if page, per_page := queryParams.Get("page"), queryParams.Get("per_page"); page != "" &&
		per_page != "" {
		page_num, err := strconv.ParseUint(page, 10, 64)
		if err != nil {
			return nil, err
		}
		per_page_num, err := strconv.ParseUint(per_page, 10, 64)
		if err != nil {
			return nil, err
		}
		paramMap["limit"] = paramPair{" LIMIT ?, ?",
			[]interface{}{(page_num-1)*per_page_num, per_page_num}}
	}

	return paramMap, nil
}

// Serves the movies and downloads that are present from the movies
// table as a JSON object. It returns pagination settings for the
// client side paginator in the JSON object as well.
func moviesTableHandler(w http.ResponseWriter, r *http.Request) {
	if err := checkAccess(w, r); err != nil {
		glog.Error(err)
		return
	}

	// No committing any reindexes from the heartbeat in between
	// queries here
	heartbeatLocks.fileIndexLock.Lock()
	defer heartbeatLocks.fileIndexLock.Unlock()

	httpError := func(err error) {
		glog.Errorf("Error in moviesTable handler: %s", err)
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

	var fileIndexKey string
	if paramPair, ok := paramMap["where"]; ok {
		fileIndexKey = paramPair.paramArgs[0].(string) + paramPair.paramArgs[1].(string)
	}

	if count, ok := fileIndexCount[fileIndexKey]; ok {
		total_entries = count
	} else {
		// We need to run a COUNT(*) query. We only need the
		// WHERE param arg
		glog.V(infoLevel).Info("Running COUNT(*) over the movie index")
		countRow := dbHandle.QueryRow(sqlStatements["getMovieNum"] + paramMap["where"].paramString,
			paramMap["where"].paramArgs...)
		if err := countRow.Scan(&total_entries); err != nil {
			httpError(err)
			return
		}
		// Update fileIndexCount
		fileIndexCount[fileIndexKey] = total_entries
	}


	paginationState := map[string]interface{} {
		"total_entries": total_entries,
	}
	// If there's a limit clause that's out of bounds, make the
	// offset 0 and adjust the paginationState accordingly
	if _, ok := paramMap["limit"]; ok {
		paramArgs := paramMap["limit"].paramArgs
		offset, limit := paramArgs[0].(uint64), paramArgs[1].(uint64)
		if offset >= total_entries {
			// We're out of bounds, change paramArgs[0] to
			// 0, and paginationState["page"] to 1. We
			// also need explicitly set per_page, because
			// otherwise backbone-paginator will reset it
			// incorrectly
			paramArgs[0] = 0
			paginationState["page"] = 1
			paginationState["per_page"] = limit
		}
	}

	rows, err := dbHandle.Query(sqlStatements["getMovies"] + paramMap["where"].paramString + paramMap["limit"].paramString,
		append(paramMap["where"].paramArgs, paramMap["limit"].paramArgs...)...)
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
	res, err := dbHandle.Exec(sqlStatements["addDownload"], *moviePath, filename)
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
