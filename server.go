
// A web server that lets clients download movies
package main

import (
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"path/filepath"
	"flag"
	"os"
	"runtime"
	"os/signal"
)

const (
	// The verbosity level necessary for info statements to be printed
	infolevel = 1
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

// Calls all the cleanup functions and flush the log
func cleanupServer() {
	cleanupDB()
	cleanupHeartbeat()
	glog.Flush()
}

// Monitors for interrupt signals and, upon getting one, calls
// cleanupServer before exiting
func interruptHandler() {
	interruptNotifier := make(chan os.Signal, 1)
	signal.Notify(interruptNotifier, os.Interrupt)
	go func() {
		select {
		case sig := <-interruptNotifier:
			switch sig {
			case os.Interrupt:
				glog.Warning("Server received an interrupt: calling cleanup functions")
				cleanupServer()
				os.Exit(0)
			}
		}
	}()
}

var (
	srcPath = flag.String("src-path", srcdir(), "The path of the movieserver source directory")
	moviePath = flag.String("movie-path", "", "REQUIRED: The path of the movies directory")
	port = flag.Uint64("port", 8080, "The port to listen on")
	refreshSchema = flag.Bool("refresh-schema", false, "If true, the server will drop and recreate the database schema")
)

func main() {
	// Sets some defaults and parses the flags
	flag.Lookup("v").Value.Set("1")
	flag.Lookup("v").DefValue = "1"
	flag.Lookup("alsologtostderr").Value.Set("true")
	flag.Lookup("alsologtostderr").DefValue = "true"

	flag.Parse()

	// movie-path must be set
	if *moviePath == "" {
		flag.PrintDefaults()
		glog.Error("movie-path flag must be set")
		return
	}

	// Sets up the threads and interrupt handler
	runtime.GOMAXPROCS(runtime.NumCPU())
	interruptHandler()
	// If we exit this function prematurely, we still want to run
	// cleanup
	defer cleanupServer()

	*srcPath = filepath.Clean(*srcPath)
	*moviePath = filepath.Clean(*moviePath)

	glog.V(infolevel).Info("Setting up SQL schema")
	if err := startupDB(); err != nil {
		glog.Error(err)
		return
	}
	glog.V(infolevel).Info("Starting the heartbeat")
	if err := startupHeartbeat(); err != nil {
		glog.Error(err)
		return
	}

	glog.V(infolevel).Info("Fetching html templates")
	if err := fetchTemplates("index"); err != nil {
		glog.Error(err)
		return
	}

	http.HandleFunc(mainPath, mainHandler)
	http.HandleFunc(fetchPath, fetchHandler)

	glog.V(infolevel).Infof("Listening on port %d\n", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		glog.Error(err)
		return
	}
}
