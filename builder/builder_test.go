package builder

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
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

func TestReadDataRejectsBrokenFile(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yml")
	broken := filepath.Join(dir, "prod.yml")
	require.NoError(t, os.WriteFile(base, []byte("host: staging"), 0o644))
	require.NoError(t, os.WriteFile(broken, []byte("host: [production"), 0o644))
	_, err := readData([]string{base, broken})
	require.ErrorContains(t, err, broken)
}

func TestRawSecretMissingPropHidesValues(t *testing.T) {
	vault.Dummy = true
	_, err := rawSecret("secret/db.passwrod")
	require.ErrorContains(t, err, "password")
	require.NotContains(t, err.Error(), "dummy-password")
}

func writeOne(t *testing.T, target, content string, perm os.FileMode) {
	t.Helper()
	source := t.TempDir()
	tpl := filepath.Join(source, "out.conf")
	require.NoError(t, os.WriteFile(tpl, []byte(content), perm))
	require.NoError(t, os.Chmod(tpl, perm))
	r, err := processFolder(source, map[string]any{})
	require.NoError(t, err)
	require.NoError(t, WriteProcessingResult(target, r))
}

func TestWriteProcessingResultEnforcesMode(t *testing.T) {
	target := t.TempDir()
	out := filepath.Join(target, "out.conf")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o644))

	writeOne(t, target, "secret", 0o600)
	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	// a read-only template must not block the next build
	writeOne(t, target, "first", 0o400)
	writeOne(t, target, "second", 0o400)
	got, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, "second", string(got))
	entries, err := os.ReadDir(target)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temp files may be left behind")
}

func TestWriteProcessingResultIgnoresUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)
	target := t.TempDir()
	writeOne(t, target, "shared", 0o644)
	info, err := os.Stat(filepath.Join(target, "out.conf"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	writeOne(t, target, "shared", 0o666)
	info, err = os.Stat(filepath.Join(target, "out.conf"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "group and others must never get write access")
}

func TestWriteProcessingResultRejectsSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	target := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(target, "values")))

	source := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(source, "values"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "values", "x.conf"), []byte("x"), 0o644))
	r, err := processFolder(source, map[string]any{})
	require.NoError(t, err)
	require.Error(t, WriteProcessingResult(target, r))
	_, err = os.Stat(filepath.Join(outside, "x.conf"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
