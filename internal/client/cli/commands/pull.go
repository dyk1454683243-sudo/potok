package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/mtiluk/potok/internal/client/config"
	"github.com/mtiluk/potok/internal/client/crypto"
	"github.com/mtiluk/potok/internal/client/secrets"
	"github.com/mtiluk/potok/internal/client/transport"
	"github.com/mtiluk/potok/internal/protocol"
	"github.com/spf13/cobra"
)

func NewPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <name>",
		Short: "Download and decrypt a registered vault's files from the server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if errors.Is(err, config.ErrNotFound) {
				return errors.New(color.RedString("Not initialised, run `potok init` first"))
			}
			if err != nil {
				return err
			}

			v, ok := cfg.Vault(name)
			if !ok {
				return errors.New(color.RedString("Vault %q is not registered locally, run `potok vault-add %s` first", name, name))
			}
			storedName := v.Name

			apiKey, err := secrets.Get(secrets.APIKey)
			if err != nil {
				return errors.New(color.RedString("Error getting API key: %v", err))
			}

			passphrase, err := secrets.Get(secrets.VaultKeyName(storedName))
			if err != nil {
				return errors.New(color.RedString("No stored passphrase for vault %q: %v", storedName, err))
			}

			client := transport.New(cfg.ServerURL, apiKey)

			remote, found, err := client.FetchVault(storedName)
			if err != nil {
				return err
			}
			if !found {
				return errors.New(color.RedString("Vault %q has not been pushed to the server yet", storedName))
			}

			key := crypto.DeriveKey([]byte(passphrase), remote.KDFSalt)

			manifestCiphertext, found, err := client.FetchManifest(storedName)
			if err != nil {
				return err
			}
			if !found {
				fmt.Println(color.YellowString("Vault %q has no manifest yet - nothing to pull.", storedName))
				return nil
			}

			manifestJSON, err := crypto.Decrypt(key, manifestCiphertext)
			if err != nil {
				return errors.New(color.RedString("Failed to decrypt manifest - wrong passphrase, or the ciphertext is corrupt: %v", err))
			}

			var manifest protocol.Manifest
			if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
				return fmt.Errorf("failed to parse decrypted manifest: %w", err)
			}

			for _, f := range manifest.Files {
				ciphertext, err := client.DownloadBlob(storedName, f.BlobID)
				if err != nil {
					return fmt.Errorf("failed to download %s: %w", f.Path, err)
				}

				plaintext, err := crypto.Decrypt(key, ciphertext)
				if err != nil {
					return errors.New(color.RedString("Failed to decrypt %s - wrong passphrase, or the blob is corrupt: %v", f.Path, err))
				}

				dest := filepath.Join(v.Path, f.Path)
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %w", f.Path, err)
				}
				if err := os.WriteFile(dest, plaintext, 0o644); err != nil {
					return fmt.Errorf("failed to write %s: %w", f.Path, err)
				}
			}

			fmt.Println(color.GreenString("Pulled %d file(s) into vault %q.", len(manifest.Files), storedName))
			return nil
		},
	}
}
