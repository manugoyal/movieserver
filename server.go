// A web server that lets clients download movies
package main

import (
        "fmt"
        "log"
        "net/http"
        "html/template"
        "path/filepath"
        "os"
        "flag"
        "io"
        "time"
        "strings"
)


/* Utility stuff */
const (
        port = ":8080"
)
var (
        moviePath = flag.String("movie-path", "movies", "The path of the movies directory")
        movieNames []string
        allowedIP = map[string]bool{"[::1]": true, "98.236.150.191": true, "174.51.196.185": true}
)

// Walks through the moviePath directory and appends any movie file
// names to movieNames, styling the movie name correctly
func movieWalkFn(path string, info os.FileInfo, err error) error {
        if err != nil {
                log.Fatal(err)
        }
        if name := info.Name(); !info.IsDir() && name[0] != '.' {
		ext := filepath.Ext(path)
		unstyled := path[len(*moviePath):len(path)-len(ext)]
		words := strings.Split(unstyled, "_")
		styled := strings.Title(strings.Join(words, " "))
                movieNames = append(movieNames, styled + ext)
        }
        return nil
}

/* HTTP  templates */

const (
        templatePath = "templates/"
        templateExt = ".html"
)
var pageTemplates = make(map[string]*template.Template)

// Gets the templates in advance, so that we don't have to repeatedly
// parse the file
func fetchTemplates(names ...string) (error) {
        for _, name := range(names) {
                t, err := template.ParseFiles(templatePath + name + templateExt)
                if err != nil {
                        return err
                }
                pageTemplates[name] = t
        }
        return nil
}

// Executes the given template and handles any errors
func runTemplate(operationName string, w http.ResponseWriter, data interface{}) {
        t, ok := pageTemplates[operationName]
        if !ok {
                panic(fmt.Sprintf("Template %s doesn't exist", operationName))
        }

        err := t.Execute(w, data)
        if err != nil {
                log.Fatal(err)
        }

        log.Printf("Rendered template %s", operationName)
}

/* HTTP handlers */

const (
        imagePath = "images/"
        mainPath = "/"
        fetchPath = "/fetch/"
)

// Makes sure that the request's ip is allowed. Sends an error message
// if it isn't. Returns true if it is allowed, false if it isn't
func checkAccess(w http.ResponseWriter, r *http.Request) bool {
        ipstr := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
        if _, ok := allowedIP[ipstr]; !ok {
                http.Error(w, fmt.Sprint("You do not have access to this site"), http.StatusServiceUnavailable)
                return false
        }
        return true
}
// If the URL is empty (just "/"), then it reindexes the movie
// directory and serves index template with the movie names.
// Otherwise, it serves the file named by the path
func mainHandler(w http.ResponseWriter, r *http.Request) {
        if !checkAccess(w, r) {
                return
        }
        if len(r.URL.Path) == 1 {
                log.Print("Indexing movie directory")
                movieNames = make([]string, 0)
                err := filepath.Walk(*moviePath, movieWalkFn)
                if err != nil {
                        log.Fatal(err)
                }
                runTemplate("index", w, movieNames)
        } else {
                http.ServeFile(w, r, r.URL.Path[1:])
        }
}

// Serves the specified file. *moviePath should not be in the url
func fetchHandler(w http.ResponseWriter, r *http.Request) {
        if !checkAccess(w, r) {
                return
        }

        filename := r.URL.Path[len(fetchPath):]
        filelocation := *moviePath + filename
        log.Printf("Fetching file: %s", filelocation)
        f, err := os.Open(filelocation)
        if err != nil {
                http.Error(w, fmt.Sprintf("Could not serve file %s", filename), http.StatusNotFound)
                return
        }
	defer f.Close()

        rs := io.ReadSeeker(f)
        w.Header().Set("Content-Type", "binary/octet-stream")
        http.ServeContent(w, r, filename, time.Time{}, rs)
        log.Printf("Served file: %s", *moviePath + filename)
}

func main() {
        flag.Parse()
        *moviePath = filepath.Clean(*moviePath) + "/"
        log.Print("Fetching html templates")
        err := fetchTemplates("index")
        if err != nil {
                log.Fatal(err)
        }

        http.HandleFunc(mainPath, mainHandler)
        http.HandleFunc(fetchPath, fetchHandler)

        log.Printf("Listening on port %s\n", port[1:])
        http.ListenAndServe(port, nil)
}
