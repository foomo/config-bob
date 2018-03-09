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

func TestIgnore(t *testing.T) {
	exampleA := GetExample("source-a")
	ignore := getIgnore(exampleA)
	if ignore[2] != "httpd/ignore-me.txt" {
		t.Fatal("ignore file parse error")
	}
}

func TestFilesAndFolders(t *testing.T) {
	exampleA := GetExample("source-a")
	match := func(topic string, actual []string, expected []string) {
		t.Log("matching", topic, "actual", actual, "expected", expected)
		for i := range expected {
			if actual[i] != expected[i] {
				t.Fatal(topic, actual[i], "!=", expected[i])
			}
		}
	}
	ignore := getIgnore(exampleA)
	files, err := getFiles(exampleA, ignore)
	panicOnErr(err)
	match("file list missmatch", files, []string{"config.yml", "httpd/copy.txt", "httpd/ext/foo.conf", "httpd/test.conf"})
	folders, err := getFolders(exampleA, ignore)
	panicOnErr(err)
	match("folder list missmatch", folders, []string{"httpd", "httpd/ext"})
}

func TestProcess(t *testing.T) {
	vault.Dummy = true
	exampleA := GetExample("source-a")
	data := make(map[string]interface{})
	jsonBytes, err := ioutil.ReadFile(GetExample("data.json"))
	panicOnErr(err)
	panicOnErr(json.Unmarshal(jsonBytes, &data))
	r, err := processFolder(exampleA, data)
	if err != nil {
		panic(err)
	}
	for filename, processingResult := range r.Files {
		fmt.Println(filename, string(processingResult.bytes))
	}
}
