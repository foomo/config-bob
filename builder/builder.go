package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// Build
func Build(args *BuilderArgs) (result *ProcessingResult, err error) {
	fmt.Println(line)
	fmt.Println("building")
	fmt.Println("data from      :", args.DataFile)
	fmt.Println("source folders :", strings.Join(args.SourceFolders, ", "))
	fmt.Println("target folder  :", args.TargetFolder)
	fmt.Println(line)
	data, err := readData(args.DataFile)
	if err != nil {
		return nil, errors.New("could not read data from: " + args.DataFile + " :: " + err.Error())
	}
	results := []*ProcessingResult{}
	if len(args.SourceFolders) == 0 {
		return nil, errors.New("there has to be at least one source folder")
	}
	for _, sourceFolder := range args.SourceFolders {
		fmt.Println(line)
		fmt.Println("processing folder", sourceFolder)
		fmt.Println(line)

		result, err := processFolder(sourceFolder, data)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	result = results[0]
	for _, r := range results[1:] {
		result.Merge(r)
	}
	return result, nil
}

const line = "-------------------------------------------------------------------------------"

func WriteProcessingResult(targetFolder string, result *ProcessingResult) error {
	fmt.Println(line)
	fmt.Println("building folder structure:")
	fmt.Println(line)
	err := os.MkdirAll(targetFolder, 0744)
	if err != nil {
		return errors.New("could not create target folder")
	}
	i := 0
	for _, folder := range result.Folders {
		i++
		folder = path.Join(targetFolder, folder)
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
	for file, processingResult := range result.Files {
		i++
		file = path.Join(targetFolder, file)
		fmt.Println(processingResult.info.Mode().Perm(), i, file)
		err := ioutil.WriteFile(file, processingResult.bytes, processingResult.info.Mode().Perm())
		if err != nil {
			return err
		}
	}
	return nil
}

func readData(file string) (data interface{}, err error) {
	if len(file) == 0 {
		return nil, nil
	}
	dataBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.New("could not read data file: " + err.Error())
	}
	d := make(map[string]interface{})
	if strings.HasSuffix(file, ".json") {
		err = json.Unmarshal(dataBytes, &d)
	} else if strings.HasSuffix(file, ".yml") || strings.HasSuffix(file, ".yaml") {
		err = yaml.Unmarshal(dataBytes, &d)
	} else {
		return nil, errors.New("unsupported data file format i need .json, .yml or .yaml")
	}
	return d, err
}

func getCopy(root string) (copy []string) {
	return getStuff(root, ".bobcopy")
}

func getStuff(root, name string) []string {
	stuff := []string{}
	stuffFile := path.Join(root, name)
	stuffBytes, err := ioutil.ReadFile(stuffFile)
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

func getIgnore(root string) (ignore []string) {
	ignore = []string{".bobignore", ".bobcopy"}
	return append(ignore, getStuff(root, ".bobignore")...)
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
	files := []string{}
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
