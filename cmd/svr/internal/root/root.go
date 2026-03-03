package root

import (
	"github.com/spf13/cobra"

	"github.com/horonlee/servora/cmd/svr/internal/cmd/gen"
)

var rootCmd = &cobra.Command{
	Use:   "svr",
	Short: "Servora development toolkit",
	Long:  "svr is the CLI toolkit for Servora.",
}

func init() {
	gen.Register(rootCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
