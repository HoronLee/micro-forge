package gen

import (
	"github.com/spf13/cobra"
)

// Register adds the gen command group to the parent command.
func Register(parent *cobra.Command) {
	genCmd := &cobra.Command{
		Use:   "gen",
		Short: "Code generation commands",
	}
	genCmd.AddCommand(NewGormCmd())
	parent.AddCommand(genCmd)
}
