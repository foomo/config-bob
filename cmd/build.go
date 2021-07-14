package cmd

import (
	"github.com/foomo/config-bob/pkg/build"
	"github.com/foomo/config-bob/pkg/providers"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringArrayVarP(&valueFiles, "value", "v", nil, "Values for templates")
	buildCmd.Flags().StringArrayVarP(&templatePaths, "template", "t", nil, "Template files and directories")
	buildCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output for generated files")
}

var (
	valueFiles    []string
	templatePaths []string
	outputPath    string
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build specified files & directories",
	Long:  `Builds, mixes in secrets etc etc...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		l := zap.L()

		manager, err := providers.NewSecretProviderManagerFromEnv(l)
		if err != nil {
			return err
		}

		cnf := build.Configuration{
			ValueFiles:    valueFiles,
			TemplatePaths: templatePaths,
			SecretManager: manager,
		}
		result, err := build.Build(l, cnf)
		if err != nil {
			return err
		}

		return build.Write(outputPath, result)
	},
}
