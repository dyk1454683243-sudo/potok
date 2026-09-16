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

func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the server's health and stored key",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Let's check the health of your server and client...")
			fmt.Println()

			config, err := config.Load()
			if err != nil {
				return errors.New(color.RedString("Error loading config: %v", err))
			}

			fmt.Println(color.GreenString("✓") + " Config loaded successfully")
			fmt.Println()

			apiKey, err := secrets.Get(secrets.APIKey)
			if err != nil {
				return errors.New(color.RedString("Error getting API key: %v", err))
			}

			client := transport.New(config.ServerURL, apiKey)

			response, err := client.Request(http.MethodGet, "/health")
			if err != nil {
				return err
			}
			defer response.Body.Close()

			fmt.Println(color.GreenString("✓") + " Health check passed")
			fmt.Println()

			meResponse, err := client.Request(http.MethodGet, "/me")
			if err != nil {
				return err
			}
			defer meResponse.Body.Close()

			if meResponse.StatusCode == http.StatusUnauthorized {
				return errors.New(color.RedString("✗") + " Invalid API Key: " + meResponse.Status)
			}

			fmt.Println(color.GreenString("✓") + " Me check passed")
			fmt.Println()

			fmt.Println("All checks passed! You're good to go!")

			return nil
		},
	}
}
