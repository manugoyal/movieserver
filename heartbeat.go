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
	// The info level for heartbeat statements needs to be higher
	// since they generate quite a bit of output
	heartbeatInfoLevel = 2
	numTasks           = 1
)

var (
	killTask       = make(chan bool, numTasks)
	heartbeatWG    sync.WaitGroup
	heartbeatLocks struct {
		// When executing separate select queries over the
		// movie entries, we sometimes don't want the size to
		// change in between queries
		fileIndexLock sync.Mutex
	}
)

// Reindexes the movies directory, setting not present to any movie
// that isn't in the current list, and adding any new movies
func indexMovies() error {
	glog.V(heartbeatInfoLevel).Infof("Movie Indexer: indexing %s", *moviePath)

	movieNames := make([]interface{}, 0)
	// Walks through the moviePath directory and appends any movie
	// file names to movieNames
	movieWalkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if name := info.Name(); !info.IsDir() && name[0] != '.' {
			movieNames = append(movieNames, path[len(*moviePath)+1:])
		}
		return nil
	}

	err := filepath.Walk(*moviePath, movieWalkFn)
	if err != nil {
		return err
	}

	// Sets all movies to non-present, then adds all the movies in
	// moviePath, which should also set existing movies back to
	// present=TRUE. It does this in one transaction, so it
	// doesn't screw up current the present table count

	trans, err := dbHandle.Begin()
	if err != nil {
		return err
	}

	if _, err = trans.Exec("UPDATE movies SET present=FALSE"); err != nil {
		trans.Rollback()
		return err
	}

	// Adds all the movies in movieNames, existing movies should
	// get set to present=True
	for _, name := range movieNames {
		if _, err = trans.Exec(sqlStatements["newMovie"], *moviePath, name); err != nil {
			trans.Rollback()
			return err
		}
	}

	// Takes the fileIndex lock when updating the movie entry set
	heartbeatLocks.fileIndexLock.Lock()
	defer heartbeatLocks.fileIndexLock.Unlock()
	if err := trans.Commit(); err != nil {
		return err
	}
	// Clears the fileIndexCount
	fileIndexCount = make(map[string]uint64)

	return nil
}

// Runs the given task continuously after sleeping for the given
// interval and logs any errors. Returns when it finds a value on the
// channel
func runTask(hfunc func() error, hname string, interval time.Duration) {
	for {
		select {
		case <-killTask:
			glog.V(heartbeatInfoLevel).Infof("Exiting %s", hname)
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
	glog.V(infoLevel).Info("Cleaning up the heartbeat")
	for i := 0; i < numTasks; i++ {
		killTask <- true
	}
	heartbeatWG.Wait()
}
