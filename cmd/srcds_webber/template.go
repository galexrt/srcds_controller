/*
Copyright 2020 Alexander Trost <galexrt@googlemail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//go:generate go get -u github.com/GeertJohan/go.rice/rice
//go:generate rice embed-go -i github.com/galexrt/srcds_controller/cmd/srcds_webber

package main

import (
	"html/template"
	"io"
	"log"
	"os"

	rice "github.com/GeertJohan/go.rice"
	"github.com/labstack/echo/v4"
)

// Template Templating
type Template struct {
	T *template.Template
}

// Render render templates
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.T.ExecuteTemplate(w, name, data)
}

func getTemplateRenderer() *Template {
	templatesBox := rice.MustFindBox("../../templates")
	templates := template.New("templates")
	templatesBox.Walk("", func(p string, i os.FileInfo, e error) error {
		if i.IsDir() {
			return nil
		}
		s, e := templatesBox.String(p)
		if e != nil {
			log.Fatalf("Failed to load template: %s\n%s\n", p, e)
		}
		template.Must(templates.New(p).Parse(s))
		return nil
	})

	return &Template{
		T: templates,
	}
}

// Page page tempalte info
type Page struct {
	Title    string
	UserInfo *UserInfo
}
