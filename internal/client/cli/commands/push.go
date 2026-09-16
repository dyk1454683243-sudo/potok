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
	"github.com/mtiluk/potok/internal/client/vault"
	"github.com/mtiluk/potok/internal/protocol"
	"github.com/spf13/cobra"
)

func NewPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push <name>",
		Short: "Encrypt and upload a registered vault's files to the server",
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

			apiKey, err := secrets.Get(secrets.APIKey)
			if err != nil {
				return errors.New(color.RedString("Error getting API key: %v", err))
			}

			passphrase, err := secrets.Get(secrets.VaultKeyName(name))
			if err != nil {
				return errors.New(color.RedString("No stored passphrase for vault %q: %v", name, err))
			}

			client := transport.New(cfg.ServerURL, apiKey)

			remote, err := client.GetOrCreateVault(name)
			if err != nil {
				return err
			}

			key := crypto.DeriveKey([]byte(passphrase), remote.KDFSalt)

			files, err := vault.ScanVault(v.Path)
			if err != nil {
				return fmt.Errorf("failed to scan vault: %w", err)
			}

			manifest := protocol.Manifest{Files: make([]protocol.ManifestFile, 0, len(files))}

			for _, f := range files {
				plaintext, err := os.ReadFile(filepath.Join(v.Path, f.Path))
				if err != nil {
					return fmt.Errorf("failed to read %s: %w", f.Path, err)
				}

				ciphertext, err := crypto.Encrypt(key, plaintext)
				if err != nil {
					return fmt.Errorf("failed to encrypt %s: %w", f.Path, err)
				}

				blobID, err := client.UploadBlob(name, ciphertext)
				if err != nil {
					return fmt.Errorf("failed to upload %s: %w", f.Path, err)
				}

				manifest.Files = append(manifest.Files, protocol.ManifestFile{
					Path:    f.Path,
					BlobID:  blobID,
					Size:    f.Size,
					ModTime: f.ModTime,
				})
			}

			manifestJSON, err := json.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("failed to encode manifest: %w", err)
			}

			manifestCiphertext, err := crypto.Encrypt(key, manifestJSON)
			if err != nil {
				return fmt.Errorf("failed to encrypt manifest: %w", err)
			}

			generation, err := client.CurrentManifestGeneration(name)
			if err != nil {
				return err
			}

			if err := client.PutManifest(name, generation, manifestCiphertext); err != nil {
				if errors.Is(err, transport.ErrConflict) {
					return errors.New(color.RedString(transport.ErrConflict.Error()))
				}
				return err
			}

			fmt.Println(color.GreenString("Pushed %d file(s) from vault %q.", len(files), name))
			return nil
		},
	}
}
