package new

import (
	"github.com/spf13/cobra"
)

// Register adds the new command group to the parent command.
func Register(parent *cobra.Command) {
	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold new components",
	}
	newCmd.AddCommand(NewApiCmd())
	parent.AddCommand(newCmd)
}
