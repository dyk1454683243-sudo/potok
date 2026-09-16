package commands

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/fatih/color"
	"github.com/mtiluk/potok/internal/client/config"
	"github.com/mtiluk/potok/internal/client/secrets"
	"github.com/mtiluk/potok/internal/client/transport"
	"github.com/spf13/cobra"
)

func NewRemoteListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remote-list",
		Short: "List all remote vaults",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.New(color.RedString("Error loading config: %v", err))
			}

			apiKey, err := secrets.Get(secrets.APIKey)
			if err != nil {
				return errors.New(color.RedString("Error getting API key: %v", err))
			}

			var vaults []transport.Vault
			response, err := transport.New(cfg.ServerURL, apiKey).RequestJSON(http.MethodGet, "/vaults", &vaults)
			if err != nil {
				return err
			}

			if response.StatusCode == http.StatusUnauthorized {
				return errors.New(color.RedString("Invalid or expired API key: %s", response.Status))
			}

			if response.StatusCode != http.StatusOK {
				return errors.New(color.RedString("Failed to list vaults: %s", response.Status))
			}

			for _, vault := range vaults {
				fmt.Println(vault.Name)
			}

			return nil
		},
	}
}
