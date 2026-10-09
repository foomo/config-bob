package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
	"text/template"
	"time"
)

const vaultServerConfigTemplate = `
backend "file" {
  path = "db"
}

listener "tcp" {
  address     = "{{.address}}"
  tls_disable = 1
}
`

type folders struct {
	db string
}
type files struct {
	conf string
	pid  string
}

type layout struct {
	folders folders
	files   files
}

func localGetLayout(folder string) layout {
	return layout{
		folders: folders{
			db: path.Join(folder, "db"),
		},
		files: files{
			conf: path.Join(folder, "config.hcl"),
			pid:  path.Join(folder, ".pid"),
		},
	}
}

func getLocalVaultAddress() string {
	return "http://" + vaultAddr
}

func LocalSetEnv() {
	os.Setenv("VAULT_ADDR", getLocalVaultAddress())
	fmt.Println("setting environment variable VAULT_ADDR:", getLocalVaultAddress())
}

func LocalSetup(folder string) error {
	l := localGetLayout(folder)
	err := os.MkdirAll(l.folders.db, 0o744)
	if err != nil {
		return err
	}
	templateData := make(map[string]string)
	templateData["address"] = vaultAddr
	t, err := template.New("temp").Parse(string(vaultServerConfigTemplate))
	if err != nil {
		return err
	}
	out := bytes.NewBuffer([]byte{})
	err = t.Execute(out, templateData)
	if err != nil {
		return err
	}
	return os.WriteFile(l.files.conf, out.Bytes(), 0o644)
}

// localStartTimeout bounds how long LocalStart waits for the server to answer
var localStartTimeout = 10 * time.Second

var localClient = &http.Client{Timeout: 5 * time.Second}

func LocalStart(folder string) (*exec.Cmd, chan error, error) {
	chanVaultErr := make(chan error, 1)
	cmd := exec.Command("vault", "server", "-config", "config.hcl")
	cmd.Dir = folder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("starting vault server with config.hcl in directory", folder)
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	go func() {
		chanVaultErr <- cmd.Wait()
	}()
	deadline := time.After(localStartTimeout)
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	for {
		select {
		case runErr := <-chanVaultErr:
			return nil, nil, fmt.Errorf("vault server exited before it was ready: %v", runErr)
		case <-deadline:
			_ = cmd.Process.Kill()
			return nil, nil, fmt.Errorf("vault server did not start within %s", localStartTimeout)
		case <-ticker.C:
			if LocalIsRunning() {
				return cmd, chanVaultErr, nil
			}
			fmt.Println("waiting for vault to start")
		}
	}
}

// LocalOpenCopy starts and unseals the vault of folder on a temporary copy, because a starting vault
// rewrites its keyring files and the committed storage must stay untouched; stop kills it and removes the copy
func LocalOpenCopy(folder string, keys []string) (stop func(), err error) {
	if LocalIsRunning() {
		// unsealing and reading would hit that other vault
		return nil, fmt.Errorf("a vault is already listening on %s, stop it first", os.Getenv("VAULT_ADDR"))
	}
	tmp, err := os.MkdirTemp("", "config-bob-vault-")
	if err != nil {
		return nil, err
	}
	if err := os.CopyFS(tmp, os.DirFS(folder)); err != nil {
		_ = os.RemoveAll(tmp)
		return nil, err
	}
	cmd, chanVaultErr, err := LocalStart(tmp)
	if err != nil {
		_ = os.RemoveAll(tmp)
		return nil, err
	}
	stop = sync.OnceFunc(func() {
		_ = cmd.Process.Kill()
		<-chanVaultErr
		_ = os.RemoveAll(tmp)
	})
	sealed := true
	for i, key := range keys {
		if sealed, err = LocalUnseal(key); err != nil {
			stop()
			return nil, fmt.Errorf("unseal key %d: %w", i+1, err)
		}
	}
	if sealed {
		stop()
		return nil, fmt.Errorf("vault is still sealed after %d unseal keys", len(keys))
	}
	return stop, nil
}

func LocalIsRunning() bool {
	addr := os.Getenv("VAULT_ADDR")
	response, err := localClient.Get(addr + "/v1/sys/init")
	if err != nil {
		fmt.Println("Could not get vault from address " + addr)
		fmt.Println(err)
		return false
	}
	defer response.Body.Close()
	contentTypes, ok := response.Header["Content-Type"]
	return (response.StatusCode == http.StatusOK) && ok && len(contentTypes) == 1 && contentTypes[0] == "application/json"
}

// LocalUnseal submits one unseal key through the HTTP API, the vault CLI only accepts it as an argument visible in ps
func LocalUnseal(key string) (sealed bool, err error) {
	body, err := json.Marshal(map[string]string{"key": key})
	if err != nil {
		return true, err
	}
	request, err := http.NewRequest(http.MethodPut, os.Getenv("VAULT_ADDR")+"/v1/sys/unseal", bytes.NewReader(body))
	if err != nil {
		return true, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := localClient.Do(request)
	if err != nil {
		return true, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return true, fmt.Errorf("unseal failed: %s %s", response.Status, message)
	}
	status := struct {
		Sealed bool `json:"sealed"`
	}{}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		return true, err
	}
	return status.Sealed, nil
}

func LocalIsSetUp(folder string) bool {
	l := localGetLayout(folder)
	checks := map[string]bool{
		l.files.conf: false,
		l.folders.db: true,
	}
	for file, isDir := range checks {
		info, err := os.Stat(file)
		if err != nil {
			return false
		}
		if isDir && !info.IsDir() || !isDir && info.IsDir() {
			return false
		}
	}
	return true
}
