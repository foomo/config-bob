package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "config-bob",
	Short: "Config bob is a template generator with secret mixin",
	Long: `Config bob is a template generator with secret mixin.
                Complete documentation is available at https://github.com/foomo/config-bob`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello there!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
