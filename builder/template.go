package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
	"gopkg.in/yaml.v2"

	"text/template"
)

var (
	secretCacheSF   = singleflight.Group{}
	secretCacheLock = &sync.RWMutex{}
	secretCache     = map[string]string{}
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
	"env": func(name string) (v string, err error) {
		v = os.Getenv(name)
		if len(v) == 0 {
			return v, fmt.Errorf("env variable %q was empty", name)
		}
		return v, nil
	},
	"indent": func(code, indent string) (string, error) {
		lines := strings.Split(code, "\n")
		var indented []string
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
	"secret": func(key string) (string, error) {
		secretCacheLock.RLock()
		if value, ok := secretCache[key]; ok {
			secretCacheLock.RUnlock()
			return value, nil
		}
		secretCacheLock.RUnlock()
		value, err, _ := secretCacheSF.Do(key, func() (interface{}, error) {
			value, err := rawSecret(key)
			if err != nil {
				return nil, err
			}
			secretCacheLock.Lock()
			secretCache[key] = value
			secretCacheLock.Unlock()
			return value, nil
		})
		if err != nil {
			return "", err
		}

		return value.(string), nil
	},
	"replace": replace,
	"op":      onePassword,
	"absPath": filepath.Abs,
	"join":    join,
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

func replace(search string, replace string, value interface{}) (v string, err error) {
	return strings.Replace(value.(string), search, replace, -1), nil
}
