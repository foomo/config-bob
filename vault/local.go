package vault

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"syscall"
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
}

func LocalSetup(folder string) error {
	l := localGetLayout(folder)
	err := os.MkdirAll(l.folders.db, 0744)
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
	return ioutil.WriteFile(l.files.conf, out.Bytes(), 0644)
}

func LocalStart(folder string) (cmd *exec.Cmd, chanVaultErr chan error) {
	chanVaultErr = make(chan error)
	cmd = exec.Command("vault", "server", "-config", "./config.hcl")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	var runErr error
	go func() {
		cmd.Dir = folder
		runErr = cmd.Run()
		chanVaultErr <- runErr
	}()
	for {
		select {
		case <-time.After(time.Millisecond * 500):
			if runErr != nil {
				fmt.Println("waiting for vault to start")
				return nil, nil
			}
			if LocalIsRunning() {
				return cmd, chanVaultErr
			} else {
				fmt.Println("local is not running")
			}
		case <-time.After(time.Second * 3):
			return nil, nil
		}
	}
}

func LocalIsRunning() bool {
	addr := os.Getenv("VAULT_ADDR")
	response, err := http.Get(addr + "/v1/")
	if err != nil {
		return false
	}
	contentTypes, ok := response.Header["Content-Type"]
	return response.StatusCode == http.StatusNotFound && ok && len(contentTypes) == 1 && contentTypes[0] == "application/json"
}

func _LocalIsRunning() bool {
	cmd := exec.Command("vault", "status")
	err := cmd.Run()
	//fmt.Println("state:", cmd.ProcessState.ExitStatus(), err, string(combined))

	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			switch status.ExitStatus() {
			case 2:
				// sealed but up
				return true
			default:
				log.Println("vault is in status", status.ExitStatus())
			}
		}
	}

	return err == nil
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
