package cli

import (
	"errors"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagServer     string
	flagIdentity   string
	flagPassphrase string
)

var rootCmd = &cobra.Command{
	Use:   "ekvs",
	Short: "EKVS – Easy Key-Value Store CLI",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagServer, "server", "", "server address (host:port) [$EKVS_SERVER]")
	rootCmd.PersistentFlags().StringVar(&flagIdentity, "identity", "", "path to OpenSSH private key [$EKVS_IDENTITY]")
	rootCmd.PersistentFlags().StringVar(&flagPassphrase, "passphrase", "", "SSH key passphrase [$EKVS_PASSPHRASE]")

	// PersistentPreRunE performs two actions before every sub-command:
	//   1. LoadIdentity: resolves the SSH identity path (flag or env-var).
	//      The actual key is loaded lazily by each command that needs it.
	//   2. Encryption session: NewSession(signer) is NOT initialised here.
	//      Each command that needs to decrypt secret values must call
	//      NewSession after loading the identity via LoadIdentity.
	//      The first consumers of this pattern are cli_export (export.go)
	//      and cli_exec (exec.go).
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flagServer == "" {
			flagServer = os.Getenv("EKVS_SERVER")
		}
		if flagIdentity == "" {
			flagIdentity = os.Getenv("EKVS_IDENTITY")
		}
		if flagPassphrase == "" {
			flagPassphrase = os.Getenv("EKVS_PASSPHRASE")
		}
		if flagServer == "" {
			return errors.New("required flag --server (or $EKVS_SERVER) not set")
		}
		if flagIdentity == "" {
			return errors.New("required flag --identity (or $EKVS_IDENTITY) not set")
		}
		return nil
	}

	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(execCmd)
}

// Execute runs the root command with os.Args.
func Execute() error {
	return rootCmd.Execute()
}

// ExecuteWithArgs runs the root command with explicit args and output writers.
// Primarily used in tests.
func ExecuteWithArgs(args []string, out, errOut io.Writer) error {
	// Reset flag values before each run so tests are independent.
	flagServer = ""
	flagIdentity = ""
	flagPassphrase = ""
	flagOutput = ""
	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}
