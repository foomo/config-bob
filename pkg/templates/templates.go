package templates

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var DefaultTemplateFunctions = template.FuncMap{
	"substr":       substr,
	"env":          env,
	"indent":       indent,
	"yaml":         toYaml,
	"jsescape":     jsEscape,
	"json":         toJson,
	"jsonindent":   jsonIndent,
	"replace":      replace,
	"absPath":      filepath.Abs,
	"join":         join,
	"contains":     contains,
	"base64encode": base64encode,
}

func base64encode(data string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(data)), nil
}

func substr(str string, ranger string) (v string, err error) {
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
}

func env(name string) (v string, err error) {
	v = os.Getenv(name)
	if len(v) == 0 {
		return v, fmt.Errorf("env variable %q was empty", name)
	}
	return v, nil
}

func indent(code, indent string) (string, error) {
	lines := strings.Split(code, "\n")
	var indented []string
	for _, line := range lines {
		indented = append(indented, indent+line)
	}
	return strings.Join(indented, "\n"), nil
}

func toYaml(value interface{}) (v string, err error) {
	yamlBytes, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%q", value), err
	}
	return strings.Trim(string(yamlBytes), "\n"), nil
}

func jsEscape(value string) (v string, err error) {
	return template.JSEscapeString(value), nil
}

func toJson(value interface{}) (v string, err error) {
	rawJSON, jsonErr := json.Marshal(value)
	if jsonErr != nil {
		return fmt.Sprintf("%q", value), jsonErr
	}
	return string(rawJSON), nil
}

func jsonIndent(value interface{}, prefix string, indent string) (v string, err error) {
	rawJSON, err := json.MarshalIndent(value, prefix, indent)
	if err == nil {
		return fmt.Sprintf("%q", value), err
	}
	return string(rawJSON), nil
}

func join(value interface{}, separator string) (string, error) {
	switch reflect.ValueOf(value).Kind() {
	case reflect.Slice, reflect.Ptr:
		values := reflect.Indirect(reflect.ValueOf(value))

		var data []string

		for i := 0; i < values.Len(); i++ {
			v := values.Index(i).Interface()
			data = append(data, fmt.Sprint(v))
		}

		return strings.Join(data, separator), nil
	default:
		return "", fmt.Errorf("function only supports slice, not %q", reflect.TypeOf(value).String())
	}
}

func contains(slice interface{}, value string) (bool, error) {
	switch reflect.ValueOf(slice).Kind() {
	case reflect.Slice, reflect.Ptr:
		values := reflect.Indirect(reflect.ValueOf(slice))

		for i := 0; i < values.Len(); i++ {
			v := values.Index(i).Interface()
			if fmt.Sprint(v) == value {
				return true, nil
			}
		}

		return false, nil
	default:
		return false, fmt.Errorf("function only supports slice, not %q", reflect.TypeOf(slice).String())
	}
}

func replace(search string, replace string, value interface{}) (v string, err error) {
	return strings.Replace(value.(string), search, replace, -1), nil
}
