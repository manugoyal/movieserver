// A web server that lets clients download movies
package main

import (
	"log"
	"net/http"
	"path/filepath"
	"flag"
	"os"
	"runtime"
	"os/signal"
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

// Calls all the cleanup functions
func cleanupServer() {
	cleanupDB()
	cleanupHeartbeat()
}

// Monitors for interrupt signals and, upon getting one, calls
// cleanupServer before exiting
func interruptHandler() {
	interruptNotifier := make(chan os.Signal, 1)
	signal.Notify(interruptNotifier, os.Interrupt)
	go func() {
		select {
		case <-interruptNotifier:
			log.Print("Server received an interrupt: calling cleanup functions")
			cleanupServer()
			os.Exit(0)
		}
	}()
}

var (
	srcPath = flag.String("src-path", srcdir(), "The path of the movieserver source directory")
	moviePath = flag.String("movie-path", "movies", "The path of the movies directory")
	port = flag.String("port", "8080", "The port to listen on")
	refreshSchema = flag.Bool("refresh-schema", false, "If true, the server will drop and recreate the database schema")
	allowedIP = map[string]bool{"[::1]": true, "98.236.150.191": true, "174.51.196.185": true}
)

func main() {
	// Sets up the threads and interrupt handler
	runtime.GOMAXPROCS(runtime.NumCPU())
	interruptHandler()
	// If we exit this function prematurely, we still want to run
	// cleanup
	defer cleanupServer()

	flag.Parse()
	*srcPath = filepath.Clean(*srcPath)
	*moviePath = filepath.Clean(*moviePath)

	log.Print("Setting up SQL schema")
	if err := connectRoot(); err != nil {
		log.Print(err)
		return
	}
	if err := setupSchema(); err != nil {
		log.Print(err)
		return
	}
	if err := compileSQL(); err != nil {
		log.Print(err)
		return
	}

	log.Print("Starting the heartbeat")
	startHeartbeat()

	log.Print("Fetching html templates")
	if err := fetchTemplates("index"); err != nil {
		log.Print(err)
		return
	}

	http.HandleFunc(mainPath, mainHandler)
	http.HandleFunc(fetchPath, fetchHandler)

	log.Printf("Listening on port %s\n", *port)
	if err := http.ListenAndServe(":" + *port, nil); err != nil {
		log.Print(err)
		return
	}
}
