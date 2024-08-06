package template

import (
	"regexp"
	"strings"
	"text/template"
)

var nameFieldInvalidCharPattern = regexp.MustCompile("[^a-z0-9]")
var templateFuncs = template.FuncMap{
	"hyphenize": func(str string) string {
		return nameFieldInvalidCharPattern.ReplaceAllString(str, "-")
	},
}

// Execute the template given as a string and return the result as a string
func executeTemplate(templateStr string, values map[string]string) (string, error) {
	theTemplate, err := template.New("").Funcs(templateFuncs).Parse(templateStr)
	if err != nil {
		return "", err
	}
	var valueBuf strings.Builder
	if err := theTemplate.Execute(&valueBuf, values); err != nil {
		return "", err
	}
	return valueBuf.String(), nil
}
