package builder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/foomo/config-bob/vaultdummy"
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
	ts := vaultdummy.DummyVaultServerSecretEcho()
	defer ts.Close()

	exampleA := GetExample("source-a")
	match := func(topic string, a []string, b []string) {
		for i := range a {
			if a[i] != b[i] {
				t.Fatal(topic)
			}
		}
	}
	match("file list missmatch", getFiles(exampleA), []string{"config.yml", "httpd/ext/foo.conf", "httpd/test.conf"})
	match("folder list missmatch", getFolders(exampleA), []string{"httpd", "httpd/ext"})
}

func TestProcess(t *testing.T) {
	ts := vaultdummy.DummyVaultServerSecretEcho()
	defer ts.Close()

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
