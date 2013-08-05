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
)


/* Utility stuff */
const (
        port = ":8080"
)
var (
        moviePath = flag.String("movie-path", "/main/movies", "The path of the movies directory")
        movieNames = make([]string, 0)
)

// Walks through the moviePath directory and appends any movie file
// names to movieNames
func movieWalkFn(path string, info os.FileInfo, err error) error {
        if err != nil {
                log.Fatal(err)
        }
        if name := info.Name(); !info.IsDir() && name[0] != '.' &&
                filepath.Ext(name) != ".srt" {
                movieNames = append(movieNames, path[len(*moviePath):])
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

// Just serves the index template with the movie names
func mainHandler(w http.ResponseWriter, r *http.Request) {
        runTemplate("index", w, movieNames)
}

// Serves the favicon
func faviconHandler(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, imagePath + "favicon.ico")
}

// Serves the specified file moviePath should not be in the url
func fetchHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len(fetchPath):]
	log.Printf("Fetching file: %s", *moviePath + filename)
	f, err := os.Open(*moviePath + filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not fetch file %s", filename), http.StatusNotFound)
		return
	}
	rs := io.ReadSeeker(f)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, filename, time.Time{}, rs)
	log.Printf("Served file: %s", filename)
}

func main() {
        flag.Parse()
	*moviePath = filepath.Clean(*moviePath) + "/"
        log.Print("Fetching html templates")
        err := fetchTemplates("index")
        if err != nil {
                log.Fatal(err)
        }

        log.Print("Indexing movie directory")

        err = filepath.Walk(*moviePath, movieWalkFn)
        if err != nil {
                log.Fatal(err)
        }

        http.HandleFunc(mainPath, mainHandler)
        http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc(fetchPath, fetchHandler)

        log.Printf("Listening on port %s\n", port[1:])
        http.ListenAndServe(port, nil)
}
