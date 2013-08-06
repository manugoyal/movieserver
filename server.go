// A web server that lets clients download movies
package main

import (
	"log"
	"net/http"
	"path/filepath"
	"flag"
	"os"
	"runtime"
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

var (
	srcPath = flag.String("src-path", srcdir(), "The path of the movieserver source directory")
	moviePath = flag.String("movie-path", "movies", "The path of the movies directory")
	port = flag.String("port", "8080", "The port to listen on")
	refreshSchema = flag.Bool("refresh-schema", false, "If true, the server will drop and recreate the database schema")
	allowedIP = map[string]bool{"[::1]": true, "98.236.150.191": true, "174.51.196.185": true}
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
	*srcPath = filepath.Clean(*srcPath)
	*moviePath = filepath.Clean(*moviePath)

	log.Print("Setting up SQL schema")
	err := connectRoot()
	if err != nil {
		log.Fatal(err)
	}
	defer cleanupDB()
	if err = setupSchema(); err != nil {
		log.Fatal(err)
	}
	if err = compileSQL(); err != nil {
		log.Fatal(err)
	}

	log.Print("Starting the heartbeat")
	startHeartbeat()

	log.Print("Fetching html templates")
	err = fetchTemplates("index")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc(mainPath, mainHandler)
	http.HandleFunc(fetchPath, fetchHandler)

	log.Printf("Listening on port %s\n", *port)
	http.ListenAndServe(":" + *port, nil)
}
