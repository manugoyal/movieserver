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
	// The verbosity level necessary for basic info statements to
	// be printed
	infoLevel = 1
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
	cleanupHeartbeat()
	cleanupDB()
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
	unblockIPs = flag.Bool("unblock-ips", false, "If true, the server will not restrict access to the allowed IP addresses")
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

	glog.V(infoLevel).Info("Setting up SQL schema")
	if err := startupDB(); err != nil {
		glog.Error(err)
		return
	}
	glog.V(infoLevel).Info("Starting the heartbeat")
	if err := startupHeartbeat(); err != nil {
		glog.Error(err)
		return
	}

	glog.V(infoLevel).Info("Fetching html templates")
	if err := fetchTemplates("index"); err != nil {
		glog.Error(err)
		return
	}

	glog.V(infoLevel).Info("Installing handlers")
	installHandlers()

	glog.V(infoLevel).Infof("Listening on port %d\n", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		glog.Error(err)
		return
	}
}
