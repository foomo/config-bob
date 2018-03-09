package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/foomo/config-bob/vault"
)

type fileResult struct {
	info     os.FileInfo
	filename string
	bytes    []byte
}

type ProcessingResult struct {
	Folders []string
	Files   map[string]*fileResult
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
	ignore := getIgnore(folderPath)
	if len(ignore) > 2 {
		fmt.Println("found .bobignore, ignoring", strings.Join(ignore, ", "))
	}
	copiedFiles := getCopy(folderPath)
	if len(copiedFiles) > 0 {
		fmt.Println("found .bobcopy, copying", strings.Join(copiedFiles, ", "))
	}
	folders, err := getFolders(folderPath, ignore)
	if err != nil {
		return
	}
	p := &ProcessingResult{
		Folders: folders,
		Files:   map[string]*fileResult{},
	}
	files, err := getFiles(folderPath, ignore)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		run := true

		for _, copyFile := range copiedFiles {
			if strings.HasPrefix(file, copyFile) || file == copyFile {
				run = false
				break
			}
		}
		p.Files[file], err = processFile(path.Join(folderPath, file), data, run)
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

func processFile(filename string, data interface{}, run bool) (result *fileResult, err error) {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil
	}
	var byteData []byte
	if run {
		fmt.Println("processing :", filename)
		processedBytes, err := process(filename, string(fileContents), data)
		if err != nil {
			return nil, err
		}
		byteData = processedBytes
	} else {
		fmt.Println("copying    :", filename)
		byteData = fileContents
	}

	info, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	return &fileResult{
		filename: filename,
		bytes:    byteData,
		info:     info,
	}, nil

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
