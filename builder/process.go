package builder

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/foomo/config-bob/vault"
)

type ProcessingResult struct {
	Folders []string
	Files   map[string][]byte
}

func (p *ProcessingResult) Merge(otherResult *ProcessingResult) {
	for _, newFolder := range otherResult.Folders {
		if p.ContainsFolder(newFolder) == false {
			p.Folders = append(p.Folders, newFolder)
		}
	}
	for filePath, fileBytes := range otherResult.Files {
		p.Files[filePath] = fileBytes
	}
}

func (p *ProcessingResult) ContainsFolder(someFolder string) bool {
	for _, f := range p.Folders {
		if someFolder == f {
			return true
		}
	}
	return false
}

func processFolder(folderPath string, data interface{}) (result *ProcessingResult, err error) {
	p := &ProcessingResult{
		Folders: getFolders(folderPath),
		Files:   make(map[string][]byte),
	}
	for _, file := range getFiles(folderPath) {

		p.Files[file], err = processFile(path.Join(folderPath, file), data)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func rawSecret(key string) (v string, err error) {
	parts := strings.Split(key, ".")
	if len(parts) == 2 {
		secretData, err := vault.Read(parts[0])
		if err != nil {
			v = "secret retrieval error" + err.Error()
			return v, errors.New(v)
		}
		return secretData[parts[1]], nil
	}
	v = "sytax error key must be \"path/to/secret.prop\""
	return v, errors.New(v)
}

func processFile(filename string, data interface{}) (result []byte, err error) {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	t, err := template.New("temp").Funcs(template.FuncMap{
		"yaml_string": func(key string) (v string) {
			v, _ = rawSecret(key)
			return v
		},
		"secret": func(key string) (v string) {
			v, _ = rawSecret(key)
			return v
		},
		"secret_js": func(key string) (v string) {
			v, _ = rawSecret(key)
			return template.JSEscapeString(v)
		},
		"secret_json": func(key string) (v string) {
			raw, _ := rawSecret(key)
			rawJSON, jsonErr := json.Marshal(raw)
			if jsonErr != nil {
				return jsonErr.Error()
			}
			return string(rawJSON)
		},
	}).Parse(string(fileContents))
	out := bytes.NewBuffer([]byte{})
	err = t.Execute(out, data)
	return out.Bytes(), err
}
