package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <projectName> [keyName] -- <program> [args]",
	Short: "Inject secrets as env vars into a subprocess",
	// At minimum: projectName + program (cobra receives everything after "--" as regular args).
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dashIdx := cmd.ArgsLenAtDash()
		if dashIdx < 0 {
			return fmt.Errorf("use -- to separate the program from project arguments: ekvs exec <project> [key] -- <program> [args]")
		}
		toolArgs := args[:dashIdx]
		programArgs := args[dashIdx:]
		if len(toolArgs) == 0 || len(programArgs) == 0 {
			return fmt.Errorf("usage: ekvs exec <project> [key] -- <program> [args]")
		}

		signer, _, fingerprint, err := LoadIdentity(flagIdentity, flagPassphrase)
		if err != nil {
			return fmt.Errorf("load identity: %w", err)
		}

		sess, err := NewSession(signer)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}

		client := NewClient("http://" + flagServer)
		project := toolArgs[0]

		var pairs []string
		if len(toolArgs) == 2 {
			keyName := toolArgs[1]
			entry, err := client.GetSecret(signer, fingerprint, project, keyName)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					return fmt.Errorf("project or key not found")
				}
				return fmt.Errorf("fetch secret: %w", err)
			}
			plaintext, err := sess.Decrypt(entry.Value)
			if err != nil {
				return fmt.Errorf("decrypt %q: %w", entry.Key, err)
			}
			pairs = []string{entry.Key + "=" + plaintext}
		} else {
			entries, err := client.ListSecrets(signer, fingerprint, project)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					return fmt.Errorf("project not found")
				}
				return fmt.Errorf("list secrets: %w", err)
			}
			for _, entry := range entries {
				plaintext, err := sess.Decrypt(entry.Value)
				if err != nil {
					return fmt.Errorf("decrypt %q: %w", entry.Key, err)
				}
				pairs = append(pairs, entry.Key+"="+plaintext)
			}
		}

		env := append(os.Environ(), pairs...)
		program := programArgs[0]
		extraArgs := programArgs[1:]

		subCmd := exec.Command(program, extraArgs...)
		subCmd.Env = env
		subCmd.Stdin = os.Stdin
		subCmd.Stdout = os.Stdout
		subCmd.Stderr = os.Stderr

		return subCmd.Run()
	},
}
