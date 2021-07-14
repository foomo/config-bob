package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/foomo/config-bob/config"
	"github.com/foomo/config-bob/vault"
	"github.com/spf13/cobra"
)

var (
	vaultKeyStore    config.KeyStore
	useVaultKeyStore = false
)

func init() {
	rootCmd.AddCommand(vaultTree)
	rootCmd.AddCommand(vaultLocal)
}

var vaultTree = &cobra.Command{
	Use:   "vault-tree",
	Short: "List secrets in vault",
	Long:  `List all secrets in vault`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(os.Args) != 3 {
			os.Exit(1)
		}

		fmt.Println("vault tree:")
		path := strings.TrimRight(os.Args[2], "/") + "/"
		fmt.Println(path)
		err := vault.Tree(path)
		if err != nil {
			fmt.Println("failed to show tree", err)
			os.Exit(1)
		}
	},
}

var vaultLocal = &cobra.Command{
	Use:   "vault-local",
	Short: "Starts a local vault",
	Long:  `Starts a local vault`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if _, ok := os.LookupEnv("CFB_DISABLE_STORE"); !ok {
			ks, err := config.NewKeyStore()
			if err != nil {
				fmt.Println("VAULT-STORE: Could not initialize vault key store, not using vault store", err)
			} else {
				fmt.Println("VAULT-STORE: Enabled")
				useVaultKeyStore = true
				vaultKeyStore = ks
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		vaultLocalUsage := func() {
			os.Exit(1)
		}
		if len(os.Args) >= 3 {
			vaultFolder, err := filepath.Abs(os.Args[2])
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
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

			vaultCommand, chanVaultErr := vault.LocalStart(vaultFolder)

			vaultKeys := getVaultKeys(vaultFolder)
			vaultToken := getVaultToken(vaultFolder)
			_ = os.Setenv("VAULT_TOKEN", vaultToken)

			if len(vaultKeys) > 0 {
				fmt.Println("trying to unseal vault:")
			}

			for _, vaultKey := range vaultKeys {
				unsealCommand, err := vault.GetUnsealCommand(vaultKey)
				fmt.Println(unsealCommand)
				if err != nil {
					log.Fatal(err)
				}

				out, err := unsealCommand.CombinedOutput()
				if err != nil {
					fmt.Println("could not unseal vault", err, string(out))
				} else {
					fmt.Println(string(out))
					//STORE VALID CREDENTIALS FOR VAULT
					fmt.Println("VAULT-STORE: Persisting valid token/key values for vault")
					if useVaultKeyStore {
						storeErr := vaultKeyStore.Store(config.VaultCredentials{
							Path:  vaultFolder,
							Token: vaultToken,
							Keys:  vaultKeys,
						})
						if storeErr != nil {
							fmt.Println("VAULT-STORE: Error ocurred while persiting vault: ", storeErr.Error())
						}
					}
				}
			}

			var cmd *exec.Cmd
			if len(os.Args) == 3 {
				fmt.Println("launching new shell", "\""+os.Getenv("SHELL")+"\"", "with pimped environment")
				cmd = exec.Command(os.Getenv("SHELL"), "--login")
			} else {
				fmt.Println("executing given script in new shell", "\""+os.Getenv("SHELL")+"\"", "with pimped environment")
				params := []string{"--login"}
				params = append(params, os.Args[3:]...)
				cmd = exec.Command(os.Getenv("SHELL"), params...)
			}

			go func() {
				vaultRunErr := <-chanVaultErr
				_ = cmd.Process.Kill()
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
				fmt.Println("could not kill vault process:", killErr.Error())
			}

			fmt.Println("Killed vault command process with PID: ", vaultCommand.Process.Pid)

			if runErr != nil {
				os.Exit(2)
			} else {
				fmt.Println("config bob says bye, bye")
			}
		} else {
			vaultLocalUsage()
		}
	},
}

func getVaultToken(vaultFolder string) string {
	vaultToken := os.Getenv("CFB_TOKEN")
	if vaultToken != "" {
		fmt.Println("Using token from CFB_TOKEN environment variable")
		return vaultToken
	}

	if useVaultKeyStore {
		if cred, ok := vaultKeyStore.Lookup(vaultFolder); ok {
			fmt.Println("VAULT-STORE: Using token from existing vault store")
			return cred.Token
		}
	}

	vaultToken, err := speakeasy.Ask("enter vault token:")
	if err != nil {
		fmt.Println("could not read token", err)
		os.Exit(1)
	}
	if len(vaultToken) > 0 {
		fmt.Println("Using token from standard input", vaultToken)
	}

	return vaultToken
}

func getVaultKeys(vaultFolder string) (vaultKeys []string) {
	environmentKeys := os.Getenv("CFB_KEYS")
	if environmentKeys != "" {
		fmt.Println("Using key from CFB_KEYS environment variable")
		vaultKeys = strings.Split(environmentKeys, ",")
		return vaultKeys
	}
	if useVaultKeyStore {
		if cred, ok := vaultKeyStore.Lookup(vaultFolder); ok {
			fmt.Println("VAULT-STORE: Using keys from existing vault store")
			return cred.Keys
		}
	}

	fmt.Println("Enter keys to unseal, terminate with empty entry")
	keyNumber := 1
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
	return
}
