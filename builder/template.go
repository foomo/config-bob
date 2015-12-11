package builder

import (
	"encoding/json"

	"gopkg.in/yaml.v2"

	"text/template"
)

// TemplateFuncs knock yourself out - this is what builder user for templating
var TemplateFuncs = template.FuncMap{
	"yaml_string": func(value string) (v string, err error) {
		return yamlString(value)
	},
	"secret_yaml": func(key string) (v string, err error) {
		v, err = rawSecret(key)
		if err != nil {
			return key, err
		}
		return yamlString(v)
	},
	"secret": func(key string) (v string, err error) {
		v, err = rawSecret(key)
		if err != nil {
			return key, err
		}
		return v, nil
	},
	"secret_js": func(key string) (v string, err error) {
		v, err = rawSecret(key)
		if err != nil {
			return key, err
		}
		return template.JSEscapeString(v), nil
	},
	"secret_json": func(key string) (v string, err error) {
		raw, secretErr := rawSecret(key)
		if secretErr != nil {
			return key, secretErr
		}
		rawJSON, jsonErr := json.Marshal(raw)
		if jsonErr != nil {
			return key, jsonErr
		}
		return string(rawJSON), nil
	},
}

func yamlString(str string) (v string, err error) {
	yamlBytes, err := yaml.Marshal(str)
	if err != nil {
		return str, err
	}
	return string(yamlBytes), nil
}

func getTemplateFuncs(data interface{}) template.FuncMap {
	return TemplateFuncs
}
