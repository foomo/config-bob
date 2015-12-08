package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func getFiles(root string) []string {
	return filterFiles(root, func(path string, fileInfo os.FileInfo) bool {
		return !fileInfo.IsDir()
	})
}

func getFolders(root string) []string {
	return filterFiles(root, func(path string, fileInfo os.FileInfo) bool {
		return fileInfo.IsDir() && path != root
	})
}

func filterFiles(root string, filter func(path string, fileInfo os.FileInfo) bool) []string {
	files := []string{}
	prefix := fmt.Sprintf("%s%c", root, os.PathSeparator)
	filepath.Walk(root, func(path string, fileInfo os.FileInfo, err error) error {
		if filter(path, fileInfo) {
			p := strings.TrimPrefix(path, prefix)
			files = append(files, p)
		}
		return err
	})
	sort.Strings(files)
	return files
}
