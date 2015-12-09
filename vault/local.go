package vault

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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

func LocalStart(folder string) (cmd *exec.Cmd, chanVaultErr chan error, err error) {
	chanVaultErr = make(chan error)
	cmd = exec.Command("vault", "server", "-config", "./config.hcl")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	var runErr error
	currentDir, wdErr := os.Getwd()
	if wdErr != nil {
		return nil, nil, errors.New("could not get work dir: " + wdErr.Error())
	}
	go func() {
		os.Chdir(folder)
		runErr = cmd.Run()
		chanVaultErr <- runErr
	}()

	cdMinus := func(err error) error {
		cdErr := os.Chdir(currentDir)
		if cdErr != nil {
			if err != nil {
				return errors.New(fmt.Sprint("could not start vault server", err.Error(), "and I did not get back to my work dir", cdErr.Error()))
			}
			return cdErr
		}
		return err
	}
	for {
		select {
		case <-time.After(time.Millisecond * 100):
			if runErr != nil {
				return nil, nil, cdMinus(runErr)
			}
			if LocalIsRunning() {
				return cmd, chanVaultErr, cdMinus(nil)
			}
		case <-time.After(time.Second * 3):
			return nil, nil, cdMinus(errors.New("local start timed out"))
		}
	}
}

func LocalIsRunning() bool {
	_, err := CallVault("/v1/seal-status")
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
