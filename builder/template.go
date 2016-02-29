package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"text/template"
)

// TemplateFuncs knock yourself out - this is what builder user for templating
var TemplateFuncs = template.FuncMap{
	"substr": func(str string, ranger string) (v string, err error) {

		rangeParts := strings.Split(ranger, ":")
		if len(rangeParts) != 2 {
			return str, fmt.Errorf("can not parse %q", ranger)
		}

		start := 0
		end := len(str)

		convert := func(strVal string, def int) (int, error) {
			if len(strVal) == 0 {
				return def, nil
			}
			i, err := strconv.Atoi(strVal)
			if err != nil {
				return i, err
			}
			if i < 0 {
				return i, errors.New("range value must be non negative")
			}
			return i, err
		}

		start, err = convert(rangeParts[0], start)
		if err != nil {
			return str, fmt.Errorf("could not parse range start in %q", ranger)
		}

		end, err = convert(rangeParts[1], end)
		if err != nil {
			return str, fmt.Errorf("could not parse range end in %q", ranger)
		}

		max := len(str)
		if end > max {
			return str, fmt.Errorf("end out of range %q length is %q", ranger, max)
		}

		substring := str[start:end]
		return substring, nil
	},
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
	"jsescape": func(value string) (v string, err error) {
		return template.JSEscapeString(value), nil
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

func ensureNonEmpty(key string, str string, err error) (string, error) {
	if err != nil {
		return str, err
	}
	if len(str) == 0 {
		if len(key) > 0 {
			return str, fmt.Errorf("not value for key %q", key)
		}
		return str, fmt.Errorf("empty values are not allowed")
	}
	return str, nil
}

func getTemplateFuncs(data interface{}) template.FuncMap {
	return TemplateFuncs
}
