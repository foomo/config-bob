package builder

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"text/template"

	"github.com/foomo/config-bob/vault"
	"golang.org/x/sync/errgroup"
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
	maps.Copy(p.Files, otherResult.Files)
}

func (p *ProcessingResult) ContainsFolder(someFolder string) bool {
	return slices.Contains(p.Folders, someFolder)
}

func processFolder(folderPath string, data any) (result *ProcessingResult, err error) {
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

	var (
		g  errgroup.Group
		mu sync.Mutex
	)
	for _, file := range files {
		run := !isCopied(file, copiedFiles)
		g.Go(func() error {
			fr, err := processFile(path.Join(folderPath, file), data, run)
			if err != nil {
				return err
			}
			mu.Lock()
			p.Files[fr.filename] = fr
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return p, nil
}

// isCopied reports whether file is listed in .bobcopy, either directly or
// by one of its parent folders
func isCopied(file string, copiedFiles []string) bool {
	for _, copyFile := range copiedFiles {
		copyFile = strings.TrimSuffix(copyFile, "/")
		if file == copyFile || strings.HasPrefix(file, copyFile+"/") {
			return true
		}
	}
	return false
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

func processFile(filename string, data any, run bool) (result *fileResult, err error) {
	fileContents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var byteData []byte
	if run {
		fmt.Println("processing :", filename)
		byteData, err = process(filename, string(fileContents), data)
		if err != nil {
			return nil, err
		}
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

func process(templName, templ string, data any) ([]byte, error) {
	t, err := template.New(templName).Option("missingkey=error").Funcs(TemplateFuncs).Parse(templ)
	if err != nil {
		return nil, fmt.Errorf("template parsing failed: %w", err)
	}
	var out bytes.Buffer
	err = t.Execute(&out, data)
	return out.Bytes(), err
}
