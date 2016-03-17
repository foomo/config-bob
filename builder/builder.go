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
	data, err := readData(args.DataFile)
	if err != nil {
		return nil, errors.New("could not read data from: " + args.DataFile + " :: " + err.Error())
	}
	results := []*ProcessingResult{}
	if len(args.SourceFolders) == 0 {
		return nil, errors.New("there has to be at least one source folder")
	}
	for _, sourceFolder := range args.SourceFolders {
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
	for _, folder := range result.Folders {
		folder = path.Join(targetFolder, folder)
		fmt.Println(folder)
		err := os.MkdirAll(folder, 0744)
		if err != nil {
			return err
		}
	}
	fmt.Println(line)
	fmt.Println("writing files:")
	fmt.Println(line)
	for file, contents := range result.Files {
		file = path.Join(targetFolder, file)
		fmt.Println(file)
		err := ioutil.WriteFile(file, contents, 0644)
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

func getFiles(root string) (files []string, err error) {
	files = filterFiles(root, func(path string, fileInfo os.FileInfo) bool {
		tartgetInfo, e := resolve(fileInfo, path)
		if e != nil {
			err = e
		}
		return !tartgetInfo.IsDir()
	})
	return
}

func getFolders(root string) (folders []string, err error) {
	folders = filterFiles(root, func(path string, fileInfo os.FileInfo) bool {
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

func walk(root string, filter filepath.WalkFunc) (err error) {
	f, err := os.Open(root)
	fileInfos, err := f.Readdir(0)
	if err != nil {
		return
	}
	for _, fileInfo := range fileInfos {
		pathname := path.Join(root, fileInfo.Name())
		targetInfo, err := resolve(fileInfo, pathname)
		// walk func does decide what to do with the errors
		err = filter(pathname, fileInfo, err)
		if err != nil {
			return err
		}
		if targetInfo.IsDir() {
			walk(pathname, filter)
		}
	}
	return
}

func filterFiles(root string, filter func(path string, fileInfo os.FileInfo) bool) []string {
	files := []string{}
	prefix := fmt.Sprintf("%s%c", root, os.PathSeparator)
	walk(root, func(path string, fileInfo os.FileInfo, err error) error {
		fmt.Println(root, path)
		if filter(path, fileInfo) {
			p := strings.TrimPrefix(path, prefix)
			files = append(files, p)
		}
		return err
	})
	sort.Strings(files)
	fmt.Println(root, files)
	return files
}
