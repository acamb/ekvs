package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export <projectName> [keyName]",
	Short: "Decrypt and print secrets to stdout",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not yet implemented")
		return errors.New("not yet implemented")
	},
}
