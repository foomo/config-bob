package builder

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

func Build(args *Args) (*ProcessingResult, error) {
	fmt.Println(line)
	fmt.Println("building")
	fmt.Println("data files     :", strings.Join(args.DataFiles, ", "))
	fmt.Println("source folders :", strings.Join(args.SourceFolders, ", "))
	fmt.Println("target folder  :", args.TargetFolder)
	fmt.Println(line)
	data, err := readData(args.DataFiles, args.ShallowMerge)
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
	// owner only: rendered files hold secrets and git checks templates out as 0644
	err := os.MkdirAll(targetFolder, 0o700)
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
		err := root.MkdirAll(folder, 0o700)
		if err != nil {
			return err
		}
	}
	fmt.Println(line)
	fmt.Println("writing files:")
	fmt.Println(line)
	i = 0
	for _, file := range slices.Sorted(maps.Keys(result.Files)) {
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

func readData(files []string, shallow bool) (any, error) {
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
			err = unmarshalYAML(dataBytes, &fileData)
		} else {
			return nil, errors.New("unsupported data file format i need .json, .yml or .yaml")
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse data file %s: %w", file, err)
		}

		if shallow {
			maps.Copy(data, fileData)
		} else {
			mergeData(data, fileData)
		}
	}
	return data, nil
}

// yaml11Bools are the YAML 1.1 booleans yaml.v2 decoded, which yaml.v3 leaves as strings
var yaml11Bools = map[string]string{
	"y": "true", "Y": "true", "yes": "true", "Yes": "true", "YES": "true", "on": "true", "On": "true", "ON": "true",
	"n": "false", "N": "false", "no": "false", "No": "false", "NO": "false", "off": "false", "Off": "false", "OFF": "false",
}

// unmarshalYAML decodes data files like yaml.v2 did, so existing templates render the same:
// YAML 1.1 booleans stay booleans and plain timestamps stay strings
func unmarshalYAML(in []byte, out any) error {
	var doc yaml.Node
	if err := yaml.Unmarshal(in, &doc); err != nil {
		return err
	}
	if doc.Kind == 0 {
		return nil
	}
	yaml11Values(&doc)
	return doc.Decode(out)
}

func yaml11Values(n *yaml.Node) {
	for i, c := range n.Content {
		// keys keep their string form, templates address them by name
		if n.Kind == yaml.MappingNode && i%2 == 0 {
			continue
		}
		if c.Kind == yaml.ScalarNode {
			plain := c.Style == 0
			if b, ok := yaml11Bools[c.Value]; ok && (plain && c.Tag == "!!str" || c.Tag == "!!bool") {
				c.Tag, c.Value = "!!bool", b
			} else if plain && c.Tag == "!!timestamp" {
				c.Tag = "!!str"
			}
		}
		yaml11Values(c)
	}
}

// mergeData deep merges src into dst: nested maps merge key by key, any other value from src replaces dst
func mergeData(dst, src any) any {
	switch s := src.(type) {
	case map[string]any:
		if d, ok := dst.(map[string]any); ok {
			for k, v := range s {
				d[k] = mergeData(d[k], v)
			}
			return d
		}
	case map[any]any:
		// yaml decodes maps with non-string keys with interface keys
		if d, ok := dst.(map[any]any); ok {
			for k, v := range s {
				d[k] = mergeData(d[k], v)
			}
			return d
		}
	}
	return src
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
			// paths are matched without a trailing slash, so "folder/" and "folder" are the same entry
			trimmedLine := strings.TrimSuffix(strings.TrimSpace(line), "/")
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

func getFiles(root string, ignore []string) ([]string, error) {
	return filterFiles(root, ignore, func(info os.FileInfo) bool {
		return !info.IsDir()
	})
}

func getFolders(root string, ignore []string) ([]string, error) {
	return filterFiles(root, ignore, func(info os.FileInfo) bool {
		return info.IsDir()
	})
}

// walk passes each entry below root to filter, symlinks resolved, and descends into the folders filter accepts
func walk(root string, filter func(path string, info os.FileInfo) (descend bool)) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		pathname := path.Join(root, entry.Name())
		info, err := os.Stat(pathname)
		if err != nil {
			return err
		}
		if filter(pathname, info) && info.IsDir() {
			if err := walk(pathname, filter); err != nil {
				return err
			}
		}
	}
	return nil
}

func filterFiles(root string, ignore []string, filter func(info os.FileInfo) bool) ([]string, error) {
	var files []string
	prefix := root + string(os.PathSeparator)
	err := walk(root, func(path string, info os.FileInfo) (descend bool) {
		if fileIsIgnored(root, path, ignore) {
			return false
		}
		if filter(info) {
			files = append(files, strings.TrimPrefix(path, prefix))
		}
		return true
	})
	sort.Strings(files)
	return files, err
}
