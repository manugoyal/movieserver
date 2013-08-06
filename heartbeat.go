// Goroutines that perform various tasks over different time intervals

package main

import (
	"path/filepath"
	"log"
	"os"
	"strings"
	"fmt"
	"time"
)


// Reindexes the movies directory, deleting from the movies table any
// movie that isn't in the current list, and adding any new movies.
func indexMovies() error {
	log.Print("Indexing movie directory")

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

	// Adds all the movies in movieNames (it does nothing if the movie already exists)
	for _, name := range(movieNames) {
		if _, err = insertStatements["newMovie"].Exec(name); err != nil {
			return err
		}
	}
	return nil
}

// Runs the given heartbeat function continuously after sleeping for
// the given interval and logs any errors
func runHeartbeat(hfunc func() error, hname string, interval time.Duration) {
	for {
		if err := hfunc(); err != nil {
			log.Printf("ERROR in %s: %s", hname, err)
		}
		time.Sleep(interval)
	}
}

// Starts each heartbeat at it's time interval
func startHeartbeat() {
	go runHeartbeat(indexMovies, "movie indexer", 5 * time.Second)
}