package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
)

// TemplateRenderer handles HTML template rendering
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewTemplateRenderer creates a new template renderer and parses all templates
func NewTemplateRenderer() (*TemplateRenderer, error) {
	tr := &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}

	if err := tr.parseTemplates(); err != nil {
		return nil, err
	}

	return tr, nil
}

// parseTemplates parses all templates from the views directory structure
func (tr *TemplateRenderer) parseTemplates() error {
	// Get all layout files
	layoutFiles, err := filepath.Glob("views/layout/*.html")
	if err != nil {
		return fmt.Errorf("error finding layout files: %w", err)
	}

	// Get all partial files
	partialFiles, err := filepath.Glob("views/partials/*.html")
	if err != nil {
		return fmt.Errorf("error finding partial files: %w", err)
	}

	// Get all page files
	pageFiles, err := filepath.Glob("views/page/*.html")
	if err != nil {
		return fmt.Errorf("error finding page files: %w", err)
	}

	// For each page, create a template that includes layouts and partials
	for _, pageFile := range pageFiles {
		// Get the base name of the page file (without extension)
		pageName := filepath.Base(pageFile)
		templateName := pageName[:len(pageName)-len(filepath.Ext(pageName))]

		// Combine all template files for this page
		var templateFiles []string
		templateFiles = append(templateFiles, layoutFiles...)
		templateFiles = append(templateFiles, partialFiles...)
		templateFiles = append(templateFiles, pageFile)

		funcMap := template.FuncMap{
			"even": func(i int) bool {
				return i%2 == 0
			},
			"subtract": func(a, b float64) float64 {
				return a - b
			},
		}

		// Parse the combined templates
		tmpl, err := template.New(templateName).Funcs(funcMap).ParseFiles(templateFiles...)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", templateName, err)
		}

		tr.templates[templateName] = tmpl
	}

	return nil
}

// Render renders a template with the given data
func (tr *TemplateRenderer) Render(w io.Writer, templateName string, data interface{}) error {
	tmpl, exists := tr.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	return tmpl.Execute(w, data)
}

// RenderToResponse renders a template directly to an HTTP response
func (tr *TemplateRenderer) RenderToResponse(w http.ResponseWriter, templateName string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tr.Render(w, templateName, data)
}

func encode[T any](w http.ResponseWriter, r *http.Request, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	if status != 200 {
		w.WriteHeader(status)
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

// Validator is an object that can be validated.
type Validator interface {
	// Valid checks the object and returns any
	// problems. If len(problems) == 0 then
	// the object is valid.
	Valid(ctx context.Context) (problems map[string]string)
}

func decodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return v, nil, nil
}
