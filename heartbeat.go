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
	"github.com/golang/glog"
	"os"
	"path/filepath"
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

// Reindexes the movies directory, setting not present to any movie
// that isn't in the current list, and adding any new movies
func indexMovies() error {
	// Sets all movies to non-present, then adds all the indexed
	// movies, which should also set existing movies back to
	// present=TRUE. It does this in one transaction, so it
	// doesn't screw up current the present table count
	trans, err := dbHandle.Begin()
	if err != nil {
		return err
	}
	if _, err = trans.Exec("UPDATE movies SET present=FALSE WHERE present=TRUE"); err != nil {
		trans.Rollback()
		return err
	}

	for _, moviePath := range moviePaths {
		glog.V(vvLevel).Infof("Movie Indexer: indexing %s", moviePath)
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
				return err
			}
			if _, err = trans.Exec(sqlStatements["newMovie"], moviePath, relpath); err != nil {
				trans.Rollback()
				return err
			}
		}
	}
	if err := trans.Commit(); err != nil {
		return err
	}
	// Clears the fileCount
	fileCount.lock.Lock()
	fileCount.index = make(map[string]uint64)
	fileCount.lock.Unlock()
	return nil
}

// Runs the given task continuously after sleeping for the given
// interval and logs any errors. Returns when it finds a value on the
// channel
func runTask(hfunc func() error, hname string, interval time.Duration) {
	for {
		select {
		case <-killTask:
			glog.V(vvLevel).Infof("Exiting %s", hname)
			heartbeatWG.Done()
			return
		default:
			if err := hfunc(); err != nil {
				glog.Errorf("%s: %s", hname, err)
			}
			time.Sleep(interval)
		}
	}
}

// Starts each task at it's time interval
func startupHeartbeat() error {
	heartbeatWG.Add(numTasks)
	go runTask(indexMovies, "Movie Indexer", 5*time.Second)
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
