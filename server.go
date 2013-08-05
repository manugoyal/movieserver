// A web server that lets clients download movies
package main

import (
        "fmt"
        "log"
        "net/http"
	"html/template"
	"path/filepath"
	"os"
)


/* Utility stuff */
const (
        port = ":8080"
	moviePath = "/Volumes/Data/Movies"
)
var movieNames []string = make([]string, 0)

// Walks through the moviePath directory and appends any movie file
// names to movieNames
func movieWalkFn(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Fatal(err)
	}
	if name := info.Name(); !info.IsDir() && name[0] != '.' &&
		filepath.Ext(name) != ".srt" {
		movieNames = append(movieNames, name)
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
	mainPath = "/"
	imagePath = "images/"
)

// Just serves the index template with the movie names
func mainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Print(movieNames)
	runTemplate("index", w, movieNames)
}

// Serves the favicon
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, imagePath + "favicon.ico")
	return
}

func main() {
	log.Print("Fetching html templates")
	err := fetchTemplates("index")
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Indexing movie directory")
	err = filepath.Walk(moviePath, movieWalkFn)
	if err != nil {
		log.Fatal(err)
	}

        http.HandleFunc(mainPath, mainHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)

        log.Printf("Listening on port %s\n", port[1:])
        http.ListenAndServe(port, nil)
}
