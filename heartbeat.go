// Goroutines that perform various tasks over different time intervals

package main

import (
	"path/filepath"
	"github.com/golang/glog"
	"os"
	"strings"
	"fmt"
	"time"
	"sync"
)

const numTasks = 1

var (
	killTask = make(chan bool, numTasks)
	heartbeatWG sync.WaitGroup
)

// Reindexes the movies directory, deleting from the movies table any
// movie that isn't in the current list, and adding any new movies.
func indexMovies() error {
	glog.V(infolevel).Infof("Movie Indexer: indexing %s", *moviePath)

	movieNames := make([]interface{}, 0)
	// Walks through the moviePath directory and appends any movie file
	// names to movieNames
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

	// Deletes any movies that aren't in movieNames
	if len(movieNames) == 0 {
		_, err = dbHandle.Exec("DELETE FROM movies")
		return err
	}
	placeholderStr := strings.Repeat("?, ", len(movieNames)-1) + "?"
	if _, err = dbHandle.Exec(fmt.Sprintf("DELETE FROM movies WHERE name NOT IN (%s)", placeholderStr), movieNames...); err != nil {
		return err
	}

	// Adds all the movies in movieNames (the query does nothing
	// if the movie already exists)
	for _, name := range(movieNames) {
		if _, err = insertStatements["newMovie"].Exec(name); err != nil {
			return err
		}
	}
	return nil
}

// Runs the given task continuously after sleeping for the given
// interval and logs any errors. Returns when it finds a value on the
// channel
func runTask(hfunc func() error, hname string, interval time.Duration) {
	for {
		select {
		case <- killTask:
			glog.V(infolevel).Infof("Exiting %s", hname)
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
	go runTask(indexMovies, "Movie Indexer", 5 * time.Second)
	return nil
}

// Sticks numTasks signals on the killTask channel and waits for all
// of them to signal on taskKilled
func cleanupHeartbeat() {
	glog.V(infolevel).Info("Cleaning up the heartbeat")
	for i := 0; i < numTasks; i++ {
		killTask <- true
	}
	heartbeatWG.Wait()
}
