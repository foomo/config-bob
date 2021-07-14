package build

import "os"

type Result struct {
	Folders []string
	Files   map[string]FileResult
}

type FileResult struct {
	Info os.FileInfo
	Name string
	Data []byte
}

type Results []Result

func (results Results) Merge() Result {
	result := Result{
		Files: map[string]FileResult{},
	}

	dirs := map[string]struct{}{}

	for _, r := range results {
		for _, dir := range r.Folders {
			dirs[dir] = struct{}{}
		}

		for filePath, fileResult := range r.Files {
			result.Files[filePath] = fileResult
		}

	}

	result.Folders = make([]string, 0, len(dirs))
	for dir := range dirs {
		result.Folders = append(result.Folders, dir)
	}

	return result
}
