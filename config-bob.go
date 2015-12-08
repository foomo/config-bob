package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/foomo/config-bob/builder"
	"github.com/foomo/config-bob/vault"
)

type stringList []string

func (l *stringList) String() string {
	return fmt.Sprint(*l)
}

func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "vault-remote":
			// VAULT_ADDR
			// VAULT_TOKEN
		case "vault-local":
			//flagVaultDir := flag.String("vault-dir", "vault", "vault db folder")
			if len(os.Args) == 3 {
				err := vault.StartAndInit(os.Args[1])
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
			} else {
				fmt.Println("usage: ", os.Args[0], " path/to/vault/folder")
				os.Exit(1)
			}
		case "_______LIST":
			os.Args = os.Args[1:]
			var sourceFolders stringList
			flag.Var(&sourceFolders, "f", "source folders")
			data := flag.String("data-file", "data.yml", "data source file")
			flag.Parse()
			fmt.Println("time to build:", sourceFolders, *data)
		case "build":
			builderArgs, err := builder.GetBuilderArgs(os.Args[2:])
			if err != nil {
				fmt.Println()
				fmt.Println("build usage", err.Error())
				os.Exit(1)
			} else {
				result, err := builder.Build(builderArgs)
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
				writeError := builder.WriteProcessingResult(builderArgs.TargetFolder, result)
				fmt.Println(writeError)
			}
		default:
			fmt.Println("help")
		}
	}
}
