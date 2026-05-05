package utils

import (
	"bytes"
	"text/template"
)

type TemplateProxy struct {
	Parse   func(name string, templateText string) (*template.Template, error)
	Execute func(*template.Template, *bytes.Buffer, interface{}) error
}

func DefaultTemplateProxy() TemplateProxy {
	return TemplateProxy{
		Parse: func(name string, templateText string) (*template.Template, error) {
			return template.New(name).Parse(templateText)
		},
		Execute: func(template *template.Template, buffer *bytes.Buffer, data interface{}) error {
			return template.Execute(buffer, data)
		},
	}
}
