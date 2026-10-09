package builder

import (
	"encoding/json"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
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

func TestIgnoreFolderWithTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".bobignore"), []byte("private/\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "private"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "private", "notes.txt"), []byte("do not ship"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.conf"), []byte("ok"), 0o644))
	r, err := processFolder(dir, nil)
	require.NoError(t, err)
	require.Empty(t, r.Folders)
	require.Equal(t, []string{"app.conf"}, slices.Sorted(maps.Keys(r.Files)))
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

func TestBuildReportsEveryTemplateError(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(a, "ok.txt"), []byte("fine"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(a, "one.txt"), []byte("{{ .missing }}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(a, "two.txt"), []byte("{{ .gone }}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(b, "three.txt"), []byte("{{ broken"), 0o644))
	_, err := Build(&Args{SourceFolders: []string{a, b}})
	require.Error(t, err)
	for _, name := range []string{"one.txt", "two.txt", "three.txt"} {
		require.ErrorContains(t, err, name)
	}
	require.NotContains(t, err.Error(), "ok.txt")
}

func TestDummySecretsRenderAnyProp(t *testing.T) {
	DummySecrets = true
	t.Cleanup(func() { DummySecrets = false })
	v, err := rawSecret("secret/mongo.uri")
	require.NoError(t, err)
	require.Equal(t, "dummy-secret:secret/mongo.uri", v)
	_, err = rawSecret("no-prop")
	require.Error(t, err, "malformed keys must still fail a check")
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
	_, err := readData([]string{base, broken}, false)
	require.ErrorContains(t, err, broken)
}

func TestReadDataDeepMerges(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	overlay := filepath.Join(dir, "globus-a.yaml")
	require.NoError(t, os.WriteFile(base, []byte(`
global:
  env: stage
  hosts: [a, b]
  db:
    host: mongo
    port: 27017
services:
  shop: {replicas: 2}
`), 0o644))
	require.NoError(t, os.WriteFile(overlay, []byte(`
global:
  hosts: [c]
  db:
    port: 27018
services: plain
`), 0o644))

	data, err := readData([]string{base, overlay}, false)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"global": map[string]any{
			"env":   "stage",
			"hosts": []any{"c"},
			"db":    map[string]any{"host": "mongo", "port": 27018},
		},
		"services": "plain",
	}, data)
}

func TestReadDataDeepMergesJSON(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.json")
	overlay := filepath.Join(dir, "overlay.json")
	require.NoError(t, os.WriteFile(base, []byte(`{"db": {"host": "mongo", "port": 1}}`), 0o644))
	require.NoError(t, os.WriteFile(overlay, []byte(`{"db": {"port": 2}}`), 0o644))

	data, err := readData([]string{base, overlay}, false)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"db": map[string]any{"host": "mongo", "port": float64(2)}}, data)
}

func TestReadDataShallowMerge(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	overlay := filepath.Join(dir, "overlay.yaml")
	require.NoError(t, os.WriteFile(base, []byte("db:\n  host: mongo\n  port: 1\nname: shop\n"), 0o644))
	require.NoError(t, os.WriteFile(overlay, []byte("db:\n  port: 2\n"), 0o644))

	data, err := readData([]string{base, overlay}, true)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"db": map[string]any{"port": 2}, "name": "shop"}, data)
}

func TestReadDataMergeLimits(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.json")
	overlay := filepath.Join(dir, "overlay.yaml")
	require.NoError(t, os.WriteFile(base, []byte(`{"db": {"host": "mongo", "port": 1}, "cache": {"ttl": 5}}`), 0o644))
	require.NoError(t, os.WriteFile(overlay, []byte("db:\n  port: 2\ncache: ~\n"), 0o644))

	data, err := readData([]string{base, overlay}, false)
	require.NoError(t, err)
	// json and yaml maps merge with each other; null replaces
	require.Equal(t, map[string]any{"db": map[string]any{"host": "mongo", "port": 2}, "cache": nil}, data)
}

func TestReadDataYAMLDecodesLikeYAMLv2(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.yaml")
	require.NoError(t, os.WriteFile(file, []byte(`
on: yes
off: No
quoted: "no"
str: !!str off
tagged: !!bool y
date: 2020-01-01
stamp: 2020-01-01T10:00:00Z
mode: 0755
nested:
  list: [ON, n, 'y', 2020-01-01]
  2: two
`), 0o644))

	data, err := readData([]string{file}, false)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"on": true, "off": false, "quoted": "no", "str": "off", "tagged": true,
		"date": "2020-01-01", "stamp": "2020-01-01T10:00:00Z", "mode": 493,
		"nested": map[any]any{"list": []any{true, false, "y", "2020-01-01"}, 2: "two"},
	}, data)

	for _, content := range []string{"", "# only a comment\n", "---\n"} {
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
		data, err := readData([]string{file}, false)
		require.NoError(t, err, content)
		require.Equal(t, map[string]any{}, data, content)
	}
}

func TestReadDataYAMLRejectsDuplicateKeys(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.yaml")
	require.NoError(t, os.WriteFile(file, []byte("a: 1\na: 2\n"), 0o644))
	_, err := readData([]string{file}, false)
	require.ErrorContains(t, err, `mapping key "a" already defined`)
}

func TestRawSecretMissingPropHidesValues(t *testing.T) {
	vault.Dummy = true
	_, err := rawSecret("secret/db.passwrod")
	require.ErrorContains(t, err, "password")
	require.NotContains(t, err.Error(), "dummy-password")
}

func TestRawSecretPathWithDots(t *testing.T) {
	vault.Dummy = true
	v, err := rawSecret("secret/example.com.user")
	require.NoError(t, err)
	require.Equal(t, "user-fromsecret/example.com", v)
	for _, key := range []string{"secret/db", "secret/db.", ".user"} {
		_, err := rawSecret(key)
		require.ErrorContains(t, err, "syntax error", key)
	}
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
