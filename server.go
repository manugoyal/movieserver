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

const (
	port = ":8080"
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
	refreshSchema = flag.Bool("refresh-schema", false, "If true, the server will drop and recreate the database schema")
	allowedusers = map[string]bool{"guardianxeroaznpride1": true, "manugoyaldesipride1": true, "rootroot": true}
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
	err = setupSchema()
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Starting the heartbeat")
	startHeartbeat()

	log.Print("Fetching html templates")
	err = fetchTemplates("login")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc(mainPath, mainHandler)
	http.HandleFunc(fetchPath, fetchHandler)

	log.Printf("Listening on port %s\n", port[1:])
	http.ListenAndServe(port, nil)
}
