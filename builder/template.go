package builder

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"text/template"
)

// TemplateFuncs knock yourself out - this is what builder user for templating
var TemplateFuncs = template.FuncMap{
	"env": func(value string) (v string, err error) {
		return os.Getenv(value), nil
	},
	"indent": func(code, indent string) (string, error) {
		lines := strings.Split(code, "\n")
		indented := []string{}
		for _, line := range lines {
			indented = append(indented, indent+line)
		}
		return strings.Join(indented, "\n"), nil
	},
	"yaml": func(value interface{}) (v string, err error) {
		yamlBytes, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%q", value), err
		}
		return strings.Trim(string(yamlBytes), "\n"), nil
	},
	"jsescape": func(key string) (v string, err error) {
		return template.JSEscapeString(v), nil
	},
	"json": func(value interface{}) (v string, err error) {
		rawJSON, jsonErr := json.Marshal(value)
		if jsonErr != nil {
			return fmt.Sprintf("%q", value), jsonErr
		}
		return string(rawJSON), nil
	},
	"jsonindent": func(value interface{}, prefix string, indent string) (v string, err error) {
		rawJSON, jsonErr := json.MarshalIndent(value, prefix, indent)
		if jsonErr != nil {
			return fmt.Sprintf("%q", value), jsonErr
		}
		return string(rawJSON), nil
	},

	"secret": func(key string) (v string, err error) {
		v, err = rawSecret(key)
		if err != nil {
			return key, err
		}
		return v, nil
	},
}

func getTemplateFuncs(data interface{}) template.FuncMap {
	return TemplateFuncs
}
