package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/foomo/config-bob/builder"
	"github.com/foomo/config-bob/vault"
	"github.com/foomo/htpasswd"
)

// Version constant specifies the current version of the script
const Version = "0.2.5"

const helpCommands = `
Commands:
    build           my main task
    vault-local     set up a local vault
    vault-htpasswd  update htpasswd files
    vault-tree      show a recursive listing in vault
    version         display version number
`

const (
	commandVersion    = "version"
	commandBuild      = "build"
	commandVaultLocal = "vault-local"
	commandVaultTree  = "vault-tree"
	commandHtpasswd   = "vault-htpasswd"
)

func isHelpFlag(arg string) bool {
	switch arg {
	case "--help", "-help", "-h":
		return true
	}
	return false
}



func help() {
	fmt.Println("usage:", os.Args[0], "<command>")
	fmt.Println(helpCommands)
}


func versionCommand() {
	fmt.Print(Version)
}

func vaultTreeCommand() {
	if len(os.Args) != 3 {
		fmt.Println("usage: ", os.Args[0], commandVaultTree, "path/in/vault")
		os.Exit(1)
	}
	fmt.Println("vault tree:")
	path := strings.TrimRight(os.Args[2], "/") + "/"
	fmt.Println(path)
	err := vault.Tree(path, 1)
	if err != nil {
		fmt.Println("failed to show tree", err)
		os.Exit(1)
	}
}

func htpasswdCommand() {
	htpasswdLocalUsage := func() {
		fmt.Println("usage: ", os.Args[0], commandHtpasswd, "path/to/htpasswd.yaml")
		os.Exit(1)
	}
	if len(os.Args) != 3 {
		htpasswdLocalUsage()
	}
	err := vault.WriteHtpasswdFiles(os.Args[2], htpasswd.HashBCrypt)
	if err != nil {
		fmt.Println("failed", err)
		os.Exit(1)

	}
	fmt.Println("DONE")
}

func vaultLocalCommand() {
	vaultLocalUsage := func() {
		fmt.Println("usage: ", os.Args[0], commandVaultLocal, "path/to/vault/folder")
		os.Exit(1)
	}
	if len(os.Args) >= 3 {
		if isHelpFlag(os.Args[2]) {
			vaultLocalUsage()
		}
		vaultFolder := os.Args[2]
		vault.LocalSetEnv()
		if !vault.LocalIsSetUp(vaultFolder) {
			fmt.Println("setting up vault tree")
			err := vault.LocalSetup(vaultFolder)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		}
		if vault.LocalIsRunning() {
			fmt.Println("there is already a vault running aborting")
			os.Exit(1)
		}
		fmt.Println("vault not running - trying to start it")

		vaultKeys := []string{}

		keyNumber := 1
		fmt.Println("Enter keys to unseal, terminate with empty entry")
		for {
			vaultKey, err := speakeasy.Ask(fmt.Sprintf("vault key %d:", keyNumber))
			if err != nil {
				fmt.Println("vault key")
				os.Exit(1)
			}
			if len(vaultKey) == 0 {
				break
			}
			vaultKeys = append(vaultKeys, vaultKey)
			keyNumber++
		}

		vaultToken, err := speakeasy.Ask("enter vault token:")
		if err != nil {
			fmt.Println("could not read token", err)
			os.Exit(1)
		}
		if len(vaultToken) > 0 {
			fmt.Println("exporting vault token", vaultToken)
			os.Setenv("VAULT_TOKEN", vaultToken)
		}

		vaultCommand, chanVaultErr := vault.LocalStart(vaultFolder)

		if len(vaultKeys) > 0 {
			fmt.Println("trying to unseal vault:")
		}
		for _, vaultKey := range vaultKeys {
			out, err := exec.Command("vault", "unseal", vaultKey).CombinedOutput()
			if err != nil {
				fmt.Println("could not unseal vault", err, string(out))
			} else {
				fmt.Println(string(out))
			}
		}

		var cmd *exec.Cmd
		if len(os.Args) == 3 {
			log.Println("launching new shell", "\""+os.Getenv("SHELL")+"\"", "with pimped environment")
			cmd = exec.Command(os.Getenv("SHELL"), "--login")
		} else {
			log.Println("executing given script in new shell", "\""+os.Getenv("SHELL")+"\"", "with pimped environment")
			params := []string{"--login"}
			params = append(params, os.Args[3:]...)
			cmd = exec.Command(os.Getenv("SHELL"), params...)
		}

		go func() {
			vaultRunErr := <-chanVaultErr
			cmd.Process.Kill()
			fmt.Println("vault died on us")
			if vaultRunErr != nil {
				fmt.Println("vault error", vaultRunErr.Error())
			}
		}()
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		runErr := cmd.Run()
		if runErr != nil {
			fmt.Println("shell exit:", runErr.Error())
		}
		killErr := vaultCommand.Process.Kill()
		if killErr != nil {
			log.Println("could not kill vault process:", killErr.Error())
		}
		if runErr != nil {
			os.Exit(2)
		} else {
			fmt.Println("config bob says bye, bye")
		}
	} else {
		vaultLocalUsage()
	}
}

func buildCommand() {
	buildUsage := func() {
		fmt.Println(
			"usage: ",
			os.Args[0],
			commandBuild,
			"path/to/source-folder-a",
			"[ path/to/source-folder-b, ... ]",
			"[ path/to/data-file.json | data-file.yaml ]",
			"path/to/target/dir",
		)
		os.Exit(1)
	}
	if isHelpFlag(os.Args[2]) {
		buildUsage()
	}
	builderArgs, err := builder.GetBuilderArgs(os.Args[2:])
	if err != nil {
		log.Println(err.Error())
		buildUsage()
	} else {
		result, err := builder.Build(builderArgs)
		if err != nil {
			fmt.Println("a build error has occurred:", err.Error())
			os.Exit(1)
		}
		writeError := builder.WriteProcessingResult(builderArgs.TargetFolder, result)
		if writeError != nil {
			fmt.Println("could not write processing result to fs:", writeError.Error())
			os.Exit(1)
		}
	}
}

func main() {

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case commandVersion:
			versionCommand()
		case commandVaultTree:
			vaultTreeCommand()
		case commandHtpasswd:
			htpasswdCommand()
		case commandVaultLocal:
			vaultLocalCommand()
		case commandBuild:
			buildCommand()
		default:
			fmt.Println("unknown command", "\""+os.Args[1]+"\"")
			help()
		}
	} else {
		help()
	}
}
