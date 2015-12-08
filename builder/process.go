package builder

import (
	"bytes"
	"io/ioutil"
	"path"
	"text/template"
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

func processFile(filename string, data interface{}) (result []byte, err error) {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	t, err := template.New("temp").Funcs(template.FuncMap{
		"secret": func(key string) (v string) {
			return "secret for " + key
		},
	}).Parse(string(fileContents))
	out := bytes.NewBuffer([]byte{})
	err = t.Execute(out, data)
	return out.Bytes(), err
}
