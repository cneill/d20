package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"strings"
)

var funcMap = template.FuncMap{
	"inStrings":    inStrings,
	"withPath":     withPath,
	"withQuery":    withQuery,
	"withoutQuery": withoutQuery,
}

type TemplateRenderer struct {
	// mainTemplate *template.Template
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

func NewTemplateRenderer() (*TemplateRenderer, error) {
	files, err := fs.Glob(templatesContent, "templates/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("failed to locate template files: %w", err)
	}

	templates := make(map[string]*template.Template, len(files))

	for _, file := range files {
		_, fileName := filepath.Split(file)
		templateName := strings.TrimSuffix(fileName, ".gohtml")

		tpl, err := template.New("layout").
			Funcs(funcMap).
			ParseFS(templatesContent, "templates/layout.gohtml")
		if err != nil {
			return nil, fmt.Errorf("failed to parse layout template: %w", err)
		}

		tpl, err = tpl.ParseFS(templatesContent, "templates/singles.gohtml")
		if err != nil {
			return nil, fmt.Errorf("failed to parse single templates: %w", err)
		}

		pageTpl, err := tpl.ParseFS(templatesContent, file)
		if err != nil {
			return nil, fmt.Errorf("failed to load template %q: %w", templateName, err)
		}

		templates[templateName] = pageTpl
	}

	renderer := &TemplateRenderer{
		templates: templates,
		funcMap:   funcMap,
	}

	return renderer, nil
}

func (t *TemplateRenderer) ExecuteSingle(writer io.Writer, name string, data any) error {
	tpl, err := template.New(name).
		Funcs(funcMap).
		ParseFS(templatesContent, "templates/singles.gohtml")
	if err != nil {
		return fmt.Errorf("failed to parse single templates: %w", err)
	}

	if err := tpl.Execute(writer, data); err != nil {
		return fmt.Errorf("error executing single template %q: %w", name, err)
	}

	return nil
}

func (t *TemplateRenderer) ExecutePage(writer io.Writer, name string, data any) error {
	if _, ok := t.templates[name]; !ok {
		return fmt.Errorf("unknown template %q", name)
	}

	if err := t.templates[name].Execute(writer, data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return nil
}

/* Template functions */

func inStrings(search string, input []string) bool {
	for _, item := range input {
		if item == search {
			return true
		}
	}

	return false
}

func withPath(path string, input *url.URL) *url.URL {
	input.Path = path
	return input
}

func withQuery(key, value string, input *url.URL) *url.URL {
	newURL, err := url.Parse(input.String())
	if err != nil {
		panic(err)
	}

	currentQuery := newURL.Query()
	currentQuery.Add(key, value)
	newURL.RawQuery = currentQuery.Encode()

	return newURL
}

func withoutQuery(key string, input *url.URL) *url.URL {
	newURL, err := url.Parse(input.String())
	if err != nil {
		panic(err)
	}

	currentQuery := newURL.Query()
	currentQuery.Del(key)
	newURL.RawQuery = currentQuery.Encode()

	return newURL
}