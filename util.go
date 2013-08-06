// Miscellaneous utility functions

package main

import (
	"os"
	"path/filepath"
)

// Looks through all the gopaths to find a possible location for the
// movieserver source. Returns the value of movieserverExt if it
// didn't find anything.
func srcdir() string {
	gopaths := filepath.SplitList(os.ExpandEnv("$GOPATH"))
	movieserverExt := "/src/github.com/manugoyal/movieserver"
	for _, path := range(gopaths) {
		_, err := os.Stat(path + movieserverExt)
		if err == nil {
			return path + movieserverExt
		}
	}
	return movieserverExt
}

// Walks through the moviePath directory and appends any movie file
// names to movieNames, styling the movie name correctly
func movieWalkFn(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if name := info.Name(); !info.IsDir() && name[0] != '.' {
		movieNames = append(movieNames, path[len(*moviePath)+1:])
	}
	return nil
}
