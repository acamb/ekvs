package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <projectName> [keyName] -- <program> [args]",
	Short: "Inject secrets as env vars into a subprocess",
	// At minimum: projectName + "--" + program (3 args after cobra strips the "--" separator).
	// cobra passes everything after "--" as regular args, so we require at least 2.
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not yet implemented")
		return errors.New("not yet implemented")
	},
}
