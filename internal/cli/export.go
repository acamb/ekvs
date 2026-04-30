package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var flagOutput string

var exportCmd = &cobra.Command{
	Use:   "export <projectName> [keyName]",
	Short: "Decrypt and print secrets to stdout",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagOutput != "" && len(args) != 2 {
			return fmt.Errorf("--output requires a keyName argument")
		}

		signer, _, fingerprint, err := LoadIdentity(flagIdentity, flagPassphrase)
		if err != nil {
			return fmt.Errorf("load identity: %w", err)
		}

		session, err := NewSession(signer)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}

		client := NewClient("http://" + flagServer)
		project := args[0]

		if len(args) == 2 {
			keyName := args[1]
			entry, err := client.GetSecret(signer, fingerprint, project, keyName)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					return fmt.Errorf("project or key not found")
				}
				return fmt.Errorf("fetch secret: %w", err)
			}
			plaintext, err := session.Decrypt(entry.Value)
			if err != nil {
				return fmt.Errorf("decrypt %q: %w", entry.Key, err)
			}
			if flagOutput != "" {
				if err := os.WriteFile(flagOutput, []byte(plaintext), 0600); err != nil {
					return fmt.Errorf("write output file: %w", err)
				}
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", entry.Key, plaintext)
			return nil
		}

		entries, err := client.ListSecrets(signer, fingerprint, project)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return fmt.Errorf("project not found")
			}
			return fmt.Errorf("list secrets: %w", err)
		}
		for _, entry := range entries {
			plaintext, err := session.Decrypt(entry.Value)
			if err != nil {
				return fmt.Errorf("decrypt %q: %w", entry.Key, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", entry.Key, plaintext)
		}
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&flagOutput, "output", "", "write decrypted value to file path (single key only)")
}
