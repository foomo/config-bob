package builder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/foomo/config-bob/vault"
)

func getCurrentDir() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func GetExample(path string) string {
	return filepath.Join(getCurrentDir(), "..", "example", path)
}

func TestFilesAndFolders(t *testing.T) {
	exampleA := GetExample("source-a")
	match := func(topic string, a []string, b []string) {
		for i := range a {
			if a[i] != b[i] {
				t.Fatal(topic)
			}
		}
	}
	files, err := getFiles(exampleA)
	panicOnErr(err)
	match("file list missmatch", files, []string{"config.yml", "httpd/ext/foo.conf", "httpd/test.conf"})
	folders, err := getFolders(exampleA)
	panicOnErr(err)
	match("folder list missmatch", folders, []string{"httpd", "httpd/ext"})
}

func TestProcess(t *testing.T) {
	vault.VaultDummy = true
	exampleA := GetExample("source-a")
	data := make(map[string]interface{})
	jsonBytes, err := ioutil.ReadFile(GetExample("data.json"))
	panicOnErr(err)
	panicOnErr(json.Unmarshal(jsonBytes, &data))
	r, err := processFolder(exampleA, data)
	if err != nil {
		panic(err)
	}
	for filename, fileBytes := range r.Files {
		fmt.Println(filename, string(fileBytes))
	}
}
