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
	"flag"
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// The verbosity level necessary for basic info statements to
	// be printed
	vLevel = 1
	// The info level for extra verbose statements
	vvLevel      = 2
	databaseName = "movieserver"
)

// Looks through all the gopaths to find a possible location for the
// movieserver source. Returns the value of movieserverExt if it
// didn't find anything.
func srcdir() string {
	gopaths := filepath.SplitList(os.ExpandEnv("$GOPATH"))
	movieserverExt := "/src/github.com/manugoyal/movieserver"
	for _, path := range gopaths {
		_, err := os.Stat(path + movieserverExt)
		if err == nil {
			return path + movieserverExt
		}
	}
	return movieserverExt
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

type moviePathMap map[string]string

func (p *moviePathMap) String() string {
	return fmt.Sprintf("%v", (map[string]string)(*p))
}
func (p *moviePathMap) Set(s string) error {
	keyval := strings.Split(s, "=")
	if len(keyval) != 2 {
		return fmt.Errorf("Argument is not a proper key-value pair")
	}
	if strings.Index(keyval[0], "/") != -1 {
		return fmt.Errorf("Key cannot include a slash")
	}
	(*p)[keyval[0]] = keyval[1]
	return nil
}

var (
	srcPath       = flag.String("src-path", srcdir(), "The path of the movieserver source directory")
	moviePaths    = make(moviePathMap)
	port          = flag.Uint64("port", 8080, "The port to listen on")
	mysqlPort     = flag.Uint64("mysql-port", 3306, "The port to connect to MySQL on")
	refreshSchema = flag.Bool("refresh-schema", false, "If true, the server will drop and recreate the database schema")
)

// Sets everything up and listens on the given port
func startupServer() {
	// If we exit this function prematurely or get interrupted, we
	// still want to run cleanup
	interruptHandler()
	defer cleanupServer()

	*srcPath = filepath.Clean(*srcPath)
	for k, _ := range moviePaths {
		moviePaths[k] = filepath.Clean(moviePaths[k])
	}

	glog.V(vLevel).Info("Setting up SQL schema")
	if err := startupDB(); err != nil {
		glog.Error(err)
		return
	}
	glog.V(vLevel).Info("Starting the heartbeat")
	if err := startupHeartbeat(); err != nil {
		glog.Error(err)
		return
	}

	glog.V(vLevel).Info("Fetching html templates")
	if err := fetchTemplates("login"); err != nil {
		glog.Error(err)
		return
	}

	glog.V(vLevel).Info("Installing handlers")
	setupHandlers()

	glog.V(vLevel).Infof("Listening on port %d\n", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		glog.Error(err)
		return
	}
}

// Calls all the cleanup functions and flushes the log
func cleanupServer() {
	cleanupHeartbeat()
	glog.Flush()
}

func main() {
	// Adds moviePaths as an argument
	flag.Var(&moviePaths, "path", "Add a path to serve (Specify as a key-value pair [name]=[path])")
	// Sets some defaults and parses the flags
	flag.Lookup("v").Value.Set("1")
	flag.Lookup("v").DefValue = "1"
	flag.Lookup("alsologtostderr").Value.Set("true")
	flag.Lookup("alsologtostderr").DefValue = "true"

	flag.Parse()

	// moviePaths must hove at least one value
	if len(moviePaths) == 0 {
		flag.PrintDefaults()
		glog.Error("There must be at least one path argument")
		return
	}

	// Makes it utilize multiple cores
	runtime.GOMAXPROCS(runtime.NumCPU())
	// Starts the server
	startupServer()
}
