package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/foomo/config-bob/pkg/providers"
	"github.com/foomo/config-bob/pkg/templates"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	line = "-------------------------------------------------------------------------------"
)

type Configuration struct {
	ValueFiles        []string
	TemplatePaths     []string
	TemplateFunctions template.FuncMap
	SecretManager     providers.SecretProviderManager
}

func Build(l *zap.Logger, config Configuration) (Result, error) {
	templateFunctions := templates.DefaultTemplateFunctions
	if len(config.TemplateFunctions) != 0 {
		templateFunctions = config.TemplateFunctions
	}

	templateFunctions["secret"] = config.SecretManager.GetSecret

	l.Info(line)
	fmt.Println("building")
	fmt.Println("data files     :", strings.Join(config.ValueFiles, ", "))
	fmt.Println("source folders :", strings.Join(config.TemplatePaths, ", "))
	fmt.Println(line)
	data, err := readData(config.ValueFiles)
	if err != nil {
		return Result{}, errors.New("could not read data from: " + strings.Join(config.ValueFiles, ", ") + " :: " + err.Error())
	}
	var results Results

	if len(config.TemplatePaths) == 0 {
		return Result{}, errors.New("there has to be at least one source folder")
	}

	for _, sourceFolder := range config.TemplatePaths {
		fmt.Println(line)
		fmt.Println("processing folder", sourceFolder)
		fmt.Println(line)

		result, err := processFolder(sourceFolder, data, templateFunctions)
		if err != nil {
			return Result{}, err
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return Result{}, nil
	}

	return results.Merge(), nil
}

func Write(outputPath string, result Result) error {
	fmt.Println(line)
	fmt.Println("building folder structure:")
	fmt.Println(line)
	err := os.MkdirAll(outputPath, 0744)
	if err != nil {
		return errors.New("could not create target folder")
	}
	i := 0
	for _, folder := range result.Folders {
		i++
		folder = path.Join(outputPath, folder)
		fmt.Println(i, folder)
		err := os.MkdirAll(folder, 0744)
		if err != nil {
			return err
		}
	}
	fmt.Println(line)
	fmt.Println("writing files:")
	fmt.Println(line)
	i = 0
	for file, res := range result.Files {
		i++
		file = path.Join(outputPath, file)
		fmt.Println(res.Info.Mode().Perm(), i, file)
		err := os.WriteFile(file, res.Data, res.Info.Mode().Perm())
		if err != nil {
			return err
		}
	}
	return nil
}

func readData(files []string) (interface{}, error) {
	if len(files) == 0 {
		return nil, nil
	}
	data := make(map[string]interface{})

	for _, file := range files {
		fileData := make(map[string]interface{})

		dataBytes, err := os.ReadFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read data file: %q", file)
		}

		var unmarshal func(data []byte, v interface{}) error
		switch {
		case strings.HasSuffix(file, ".json"):
			unmarshal = json.Unmarshal
		case strings.HasSuffix(file, ".yml") || strings.HasSuffix(file, ".yaml"):
			unmarshal = yaml.Unmarshal
		default:
			return nil, errors.New("unsupported data file format i need .json, .yml or .yaml")
		}

		err = unmarshal(dataBytes, &fileData)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal data from file %q", file)
		}

		for k, v := range fileData {
			data[k] = v
		}
	}
	return data, nil
}

func processFolder(folderPath string, data interface{}, templateFuncMap template.FuncMap) (result Result, err error) {
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
	p := Result{
		Folders: folders,
		Files:   map[string]FileResult{},
	}
	files, err := getFiles(folderPath, ignore)
	if err != nil {
		return Result{}, err
	}
	for _, file := range files {
		run := true

		for _, copyFile := range copiedFiles {
			if strings.HasPrefix(file, copyFile) || file == copyFile {
				run = false
				break
			}
		}
		p.Files[file], err = processFile(path.Join(folderPath, file), data, run, templateFuncMap)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func getIgnore(root string) (ignore []string) {
	ignore = []string{".bobignore", ".bobcopy"}
	return append(ignore, getStuff(root, ".bobignore")...)
}

func getStuff(root, name string) []string {
	var stuff []string
	stuffFile := path.Join(root, name)
	stuffBytes, err := os.ReadFile(stuffFile)
	if err == nil {
		lines := strings.Split(string(stuffBytes), "\n")
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if len(trimmedLine) > 0 {
				stuff = append(stuff, trimmedLine)
			}
		}
	}
	return stuff
}

func fileIsIgnored(root string, p string, ignore []string) bool {
	prefix := root + string(os.PathSeparator)
	trimmedPath := strings.TrimPrefix(p, prefix)
	for _, ignored := range ignore {
		if trimmedPath == ignored {
			return true
		}
	}
	return false
}

func getFiles(root string, ignore []string) (files []string, err error) {
	files, err = filterFiles(root, ignore, func(path string, fileInfo os.FileInfo) bool {
		tartgetInfo, e := resolve(fileInfo, path)
		if e != nil {
			err = e
		}
		return !tartgetInfo.IsDir()
	})
	return
}

func getFolders(root string, ignore []string) (folders []string, err error) {
	folders, err = filterFiles(root, ignore, func(path string, fileInfo os.FileInfo) bool {
		targetInfo, e := resolve(fileInfo, path)
		if e != nil {
			err = e
		}
		return path != root && targetInfo.IsDir()
	})
	return
}

func resolve(info os.FileInfo, p string) (targetInfo os.FileInfo, err error) {
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		// let us take a look at the target
		target, err := filepath.EvalSymlinks(p)
		if err == nil {
			return os.Stat(target)
		}
		return nil, err
	}
	return info, nil
}

func walk(root string, ignore []string, filter func(path string, fileInfo os.FileInfo) (descend bool)) (err error) {
	f, err := os.Open(root)
	if err != nil {
		return err
	}

	fileInfos, err := f.Readdir(0)
	if err != nil {
		return
	}
	for _, fileInfo := range fileInfos {
		pathname := path.Join(root, fileInfo.Name())
		targetInfo, err := resolve(fileInfo, pathname)
		if err != nil {
			return err
		}
		// walk func does decide what to do with the errors
		if filter(pathname, fileInfo) {
			if targetInfo.IsDir() {
				err = walk(pathname, ignore, filter)
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}

func filterFiles(root string, ignore []string, filter func(path string, fileInfo os.FileInfo) bool) ([]string, error) {
	var files []string
	prefix := root + string(os.PathSeparator)
	err := walk(root, ignore, func(path string, fileInfo os.FileInfo) (descend bool) {
		if filter(path, fileInfo) && !fileIsIgnored(root, path, ignore) {
			p := strings.TrimPrefix(path, prefix)
			files = append(files, p)
		}
		return !fileIsIgnored(root, path, ignore)
	})
	sort.Strings(files)
	return files, err
}

func getCopy(root string) (copy []string) {
	return getStuff(root, ".bobcopy")
}

func processFile(filename string, data interface{}, run bool, templateFuncMap template.FuncMap) (result FileResult, err error) {
	fileContents, err := os.ReadFile(filename)
	if err != nil {
		return FileResult{}, nil
	}
	var byteData []byte
	if run {
		fmt.Println("processing :", filename)
		processedBytes, err := process(filename, string(fileContents), data, templateFuncMap)
		if err != nil {
			return FileResult{}, err
		}
		byteData = processedBytes
	} else {
		fmt.Println("copying    :", filename)
		byteData = fileContents
	}

	info, err := os.Stat(filename)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{
		Name: filename,
		Data: byteData,
		Info: info,
	}, nil

}

func process(templName, templ string, data interface{}, templateFuncMap template.FuncMap) (result []byte, err error) {
	t, err := template.New(templName).Option("missingkey=error").Funcs(templateFuncMap).Parse(templ)
	if err != nil {
		return
	}
	out := bytes.NewBuffer([]byte{})
	err = t.Execute(out, data)
	return out.Bytes(), err
}
