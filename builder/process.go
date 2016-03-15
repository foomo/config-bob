package builder

import (
	"bytes"
	"errors"
	"fmt"
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
	folderPath = path.Clean(folderPath)
	folders, err := getFolders(folderPath)
	if err != nil {
		return
	}
	p := &ProcessingResult{
		Folders: folders,
		Files:   make(map[string][]byte),
	}
	files, err := getFiles(folderPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {

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
			v = "secret retrieval error: " + err.Error()
			return v, errors.New(v)
		}
		prop := parts[1]
		s, ok := secretData[prop]
		if !ok {
			return "<prop not found on secret>", errors.New("property \"" + prop + "\" is not set for secret " + parts[0] + " " + fmt.Sprint(secretData))
		}
		return s, nil
	}
	v = "syntax error key must be \"path/to/secret.prop\""
	return v, errors.New(v)
}

func rawTemplate(data interface{}, key string) string {
	t, err := template.New("temp").Parse("{{ " + key + " }}")
	if err != nil {
		return key + "caused error: " + err.Error()
	}
	out := bytes.NewBuffer([]byte{})
	err = t.Execute(out, data)
	return string(out.Bytes())
}

func processFile(filename string, data interface{}) (result []byte, err error) {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return process(filename, string(fileContents), data)
}

func process(templName, templ string, data interface{}) (result []byte, err error) {
	t, err := template.New(templName).Option("missingkey=error").Funcs(TemplateFuncs).Parse(templ)
	if err != nil {
		return
	}
	out := bytes.NewBuffer([]byte{})
	err = t.Execute(out, data)
	return out.Bytes(), err
}
