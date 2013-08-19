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

// Goroutines that perform various tasks over different time intervals

package main

import (
	"fmt"
	"github.com/golang/glog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	numTasks = 1
)

var (
	killTask    = make(chan bool, numTasks)
	heartbeatWG sync.WaitGroup
)

// The indexer keeps a set of movies in the moviePaths directories in
// memory, so that reindexing and adding/deleting entries from the
// database is faster. movieMap is a map from paths to a map of names
// to bools. It should only be accessed by the indexMovies bootstrap
// and task functions.
var movieMap map[string](map[string]bool)

// Initializes movieMap to the existing entries in the database
func bootstrapIndexMovies(name string) error {
	glog.V(vvLevel).Infof("%s: bootstrapping", name)
	movieMap = make(map[string](map[string]bool))
	for _, path := range moviePaths {
		movieMap[path] = make(map[string]bool)
	}

	moviePathStr := strings.Repeat("?, ", len(moviePaths)-1) + "?"
	moviePathArgs := make([]interface{}, 0, len(moviePaths))
	for _, v := range moviePaths {
		moviePathArgs = append(moviePathArgs, v)
	}
	rows, err := dbHandle.Query(
		fmt.Sprintf("SELECT path, name FROM movies WHERE path IN (%s)", moviePathStr),
		moviePathArgs...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var path, name string
		if err := rows.Scan(&path, &name); err != nil {
			return err
		}
		movieMap[path][name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

// Reindexes the movies directory, deleting any movie in movieMap that
// wasn't encountered, and adding any new movies.
func indexMovies(name string) error {
	trans, err := dbHandle.Begin()
	if err != nil {
		return err
	}
	// Double-buffers the movieMap, so that if the transaction
	// rolls back, the movieMap isn't modified.
	innerMovieMap := make(map[string](map[string]bool))
	for _, path := range moviePaths {
		innerMovieMap[path] = make(map[string]bool)
	}
	// When copying the data over to innerMovieMap, we set all the
	// movies in the current list to false, to indicate that they
	// are to be deleted. We set all the movies we encounter in
	// the indexing to true (if it's a new movie, we add it to the
	// database with an insert query). The remaining movies that
	// are false are deleted from the map and from the database
	for path, nameMap := range movieMap {
		for name, _ := range nameMap {
			innerMovieMap[path][name] = false
		}
	}
	for _, moviePath := range moviePaths {
		glog.V(vvLevel).Infof("%s: indexing %s", name, moviePath)
		fileChan := make(chan filePair)
		go walkDir(moviePath, fileChan, func(fp filePair) bool {
			if (fp.path != moviePath && filepath.Base(fp.path)[0] == '.') ||
				fp.fi.Mode()&os.ModeSymlink > 0 {
				return false
			}
			return true
		})
		for fp := range fileChan {
			relpath, err := filepath.Rel(moviePath, fp.path)
			if err != nil {
				trans.Rollback()
				return err
			}
			_, ok := innerMovieMap[moviePath][relpath]
			if !ok {
				// Inserts the movie into the db,
				// since it wasn't in movieMap
				// originally
				if _, err := trans.Exec(sqlStatements["newMovie"], moviePath, relpath); err != nil {
					trans.Rollback()
					return err
				}
			}
			innerMovieMap[moviePath][relpath] = true
		}
	}
	// Deletes all movies in innerMovieMap that are false
	for path, innerNameMap := range innerMovieMap {
		for name, ok := range innerNameMap {
			if !ok {
				if _, err := trans.Exec(sqlStatements["deleteMovie"], path, name); err != nil {
					trans.Rollback()
					return err
				}
				delete(innerNameMap, name)

			}
		}
	}

	if err := trans.Commit(); err != nil {
		return err
	}
	movieMap = innerMovieMap
	return nil
}

type taskFunc func(string) error

// Runs the given task continuously after sleeping for the given
// interval and logs any errors. Returns when it finds a value on the
// channel
func runTask(bFunc taskFunc, tFunc taskFunc, name string, interval time.Duration) {
	if err := bFunc(name); err != nil {
		glog.Errorf("%s: %s", name, err)
	}
	for {
		select {
		case <-killTask:
			glog.V(vvLevel).Infof("Exiting %s", name)
			heartbeatWG.Done()
			return
		default:
			if err := tFunc(name); err != nil {
				glog.Errorf("%s: %s", name, err)
			}
			time.Sleep(interval)
		}
	}
}

// Starts each task at it's time interval
func startupHeartbeat() error {
	heartbeatWG.Add(numTasks)
	go runTask(bootstrapIndexMovies, indexMovies, "Movie Indexer", 5*time.Second)
	return nil
}

// Sticks numTasks signals on the killTask channel and waits for all
// of them to signal on taskKilled
func cleanupHeartbeat() {
	glog.V(vLevel).Info("Cleaning up the heartbeat")
	for i := 0; i < numTasks; i++ {
		killTask <- true
	}
	heartbeatWG.Wait()
}
