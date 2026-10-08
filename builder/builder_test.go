package builder

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/foomo/config-bob/vault"
	"github.com/stretchr/testify/require"
)

func getCurrentDir() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
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
	require.NoError(t, err)
	match("file list missmatch", files, []string{"config.yml", "httpd/copy.txt", "httpd/ext/foo.conf", "httpd/test.conf"})
	folders, err := getFolders(exampleA, ignore)
	require.NoError(t, err)
	match("folder list missmatch", folders, []string{"httpd", "httpd/ext"})
}

func TestProcess(t *testing.T) {
	vault.Dummy = true
	exampleA := GetExample("source-a")
	data := make(map[string]any)
	jsonBytes, err := os.ReadFile(GetExample("data.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(jsonBytes, &data))
	r, err := processFolder(exampleA, data)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"httpd", "httpd/ext"}, r.Folders)

	expected := map[string]string{
		"config.yml": `---
killer: |2-

  :foo
  bar
  jkljkljkljkljkl:]|
account:
  name: call my name
  password: dummy-password
payment:
  token: well-a-token
`,
		"httpd/test.conf": `<VirtualHost *:80>

	ServerName <test.local>
	# environment variables
	SetEnv FOOMO_RUN_MODE "test"

	AddOutputFilterByType DEFLATE text/html text/plain text/xml text/x-js text/css application/javascript application/x-json
</VirtualHost>
`,
		"httpd/ext/foo.conf": "# included above\n",
		// listed in .bobcopy, so it must not be rendered as a template
		"httpd/copy.txt": "{{ copy me or you will die}}",
	}

	require.Len(t, r.Files, len(expected))
	for name, content := range expected {
		fileResult, ok := r.Files[name]
		require.True(t, ok, "missing file %q", name)
		require.Equal(t, content, string(fileResult.bytes), "content mismatch for %q", name)
	}
}

func TestProcessTemplateError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("fine"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.txt"), []byte("{{ .missing }}"), 0o644))
	_, err := processFolder(dir, map[string]any{})
	require.Error(t, err)
}

func TestProcessCopyPrefix(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".bobcopy"), []byte("raw.txt"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "raw.txt"), []byte("{{ .x }}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "raw.txt.tpl"), []byte("{{ .x }}"), 0o644))
	r, err := processFolder(dir, map[string]any{"x": "rendered"})
	require.NoError(t, err)
	require.Equal(t, "{{ .x }}", string(r.Files["raw.txt"].bytes))
	require.Equal(t, "rendered", string(r.Files["raw.txt.tpl"].bytes))
}

func TestWriteProcessingResultPaths(t *testing.T) {
	source := filepath.Join(t.TempDir(), "templates")
	require.NoError(t, os.MkdirAll(filepath.Join(source, "values"), 0o755))
	tpl := filepath.Join(source, "values", "x.yaml")
	require.NoError(t, os.WriteFile(tpl, []byte("name: {{ .name }}"), 0o644))

	r, err := processFolder(source, map[string]any{"name": "demo"})
	require.NoError(t, err)
	target := t.TempDir()
	require.NoError(t, WriteProcessingResult(target, r))

	out, err := os.ReadFile(filepath.Join(target, "values", "x.yaml"))
	require.NoError(t, err)
	require.Equal(t, "name: demo", string(out))
	src, err := os.ReadFile(tpl)
	require.NoError(t, err)
	require.Equal(t, "name: {{ .name }}", string(src), "the source template must stay untouched")
}
