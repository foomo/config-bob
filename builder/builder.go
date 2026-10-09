package builder

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/samber/lo"
	"gopkg.in/yaml.v2"
)

func Build(args *Args) (*ProcessingResult, error) {
	fmt.Println(line)
	fmt.Println("building")
	fmt.Println("data files     :", strings.Join(args.DataFiles, ", "))
	fmt.Println("source folders :", strings.Join(args.SourceFolders, ", "))
	fmt.Println("target folder  :", args.TargetFolder)
	fmt.Println(line)
	data, err := readData(args.DataFiles)
	if err != nil {
		return nil, errors.New("could not read data from: " + strings.Join(args.DataFiles, ", ") + " :: " + err.Error())
	}

	var (
		results []*ProcessingResult
		errs    []error
	)

	if len(args.SourceFolders) == 0 {
		return nil, errors.New("there has to be at least one source folder")
	}
	for _, sourceFolder := range args.SourceFolders {
		fmt.Println(line)
		fmt.Println("processing folder", sourceFolder)
		fmt.Println(line)

		result, err := processFolder(sourceFolder, data)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		results = append(results, result)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if len(results) == 0 {
		return nil, nil
	}

	result := results[0]
	if len(results) > 1 {
		for _, r := range results[1:] {
			result.Merge(r)
		}
	}

	return result, nil
}

const line = "-------------------------------------------------------------------------------"

func WriteProcessingResult(targetFolder string, result *ProcessingResult) error {
	fmt.Println(line)
	fmt.Println("building folder structure:")
	fmt.Println(line)
	err := os.MkdirAll(targetFolder, 0o744)
	if err != nil {
		return errors.New("could not create target folder")
	}
	// writing through a root keeps symlinks inside the target from redirecting output outside of it
	root, err := os.OpenRoot(targetFolder)
	if err != nil {
		return err
	}
	defer root.Close()
	i := 0
	sort.Strings(result.Folders)
	for _, folder := range result.Folders {
		i++
		fmt.Println(i, path.Join(targetFolder, folder))
		err := root.MkdirAll(folder, 0o744)
		if err != nil {
			return err
		}
	}
	fmt.Println(line)
	fmt.Println("writing files:")
	fmt.Println(line)
	i = 0
	keys := lo.Keys(result.Files)
	sort.Strings(keys)

	for _, file := range keys {
		processingResult := result.Files[file]
		i++
		// keep the template mode but never let others write the output
		perm := processingResult.info.Mode().Perm() &^ 0o022
		fmt.Println(perm, i, path.Join(targetFolder, file))
		if err := replaceFile(root, file, processingResult.bytes, perm); err != nil {
			return err
		}
	}
	return nil
}

// replaceFile writes a temp file and renames it over name, so existing or read-only outputs get exactly perm
func replaceFile(root *os.Root, name string, data []byte, perm os.FileMode) error {
	tmp := name + ".bob-tmp-" + rand.Text()
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		// set the mode explicitly, OpenFile would apply the umask
		err = f.Chmod(perm)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = root.Rename(tmp, name)
	}
	if err != nil {
		_ = root.Remove(tmp)
	}
	return err
}

func readData(files []string) (any, error) {
	if len(files) == 0 {
		return nil, nil
	}
	data := make(map[string]any)

	for _, file := range files {
		fileData := make(map[string]any)

		dataBytes, err := os.ReadFile(file)
		if err != nil {
			return nil, errors.New("could not read data file: " + err.Error())
		}
		if strings.HasSuffix(file, ".json") {
			err = json.Unmarshal(dataBytes, &fileData)
		} else if strings.HasSuffix(file, ".yml") || strings.HasSuffix(file, ".yaml") {
			err = yaml.Unmarshal(dataBytes, &fileData)
		} else {
			return nil, errors.New("unsupported data file format i need .json, .yml or .yaml")
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse data file %s: %w", file, err)
		}

		maps.Copy(data, fileData)
	}
	return data, nil
}

func getCopy(root string) (copy []string) {
	return getStuff(root, ".bobcopy")
}

func getStuff(root, name string) []string {
	var stuff []string
	stuffFile := path.Join(root, name)
	stuffBytes, err := os.ReadFile(stuffFile)
	if err == nil {
		lines := strings.SplitSeq(string(stuffBytes), "\n")
		for line := range lines {
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
	return slices.Contains(ignore, trimmedPath)
}

func getFiles(root string, ignore []string) (files []string, err error) {
	files, err = filterFiles(root, ignore, func(path string, fileInfo os.FileInfo) bool {
		tartgetInfo, e := resolve(fileInfo, path)
		if e != nil {
			err = e
		}
		return !tartgetInfo.IsDir()
	})
	sort.Strings(files)
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
