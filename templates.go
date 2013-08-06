// HTTP  templates

package main 

import (
	"html/template"
	"net/http"
	"log"
	"fmt"
)


var pageTemplates = make(map[string]*template.Template)

// Gets the templates in advance, so that we don't have to repeatedly
// parse the file
func fetchTemplates(names ...string) (error) {
	for _, name := range(names) {
		t, err := template.ParseFiles(fmt.Sprintf("%s/templates/%s.html", *srcPath, name))
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

