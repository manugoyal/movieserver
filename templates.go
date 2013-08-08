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

// HTTP  templates

package main 

import (
	"html/template"
	"net/http"
	"fmt"
)


var pageTemplates = make(map[string]*template.Template)

// Gets the templates in advance, so that we don't have to repeatedly
// parse the file
func fetchTemplates(names ...string) (error) {
	for _, name := range(names) {
		t, err := template.ParseFiles(fmt.Sprintf("%s/static/templates/%s.html", *srcPath, name))
		if err != nil {
			return err
		}
		pageTemplates[name] = t
	}
	return nil
}

// Executes the given template and handles any errors
func runTemplate(operationName string, w http.ResponseWriter, data interface{}) error {
	t, ok := pageTemplates[operationName]
	if !ok {
		panic(fmt.Sprintf("Template %s doesn't exist", operationName))
	}

	err := t.Execute(w, data)
	if err != nil {
		return err
	}
	return nil
}

