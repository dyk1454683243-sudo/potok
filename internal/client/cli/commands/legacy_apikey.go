package commands

import (
	"errors"
	"fmt"

	"github.com/mtiluk/potok/internal/client/config"
	"github.com/mtiluk/potok/internal/client/secrets"
)

// MigrateLegacyAPIKey moves a leftover config.json api_key into the OS
// keyring and rewrites the file without the secret. If the keyring already
// has an API key, the file value is discarded so the keyring stays canonical.
func MigrateLegacyAPIKey() error {
	key, present, err := config.PeekLegacyAPIKey()
	if err != nil || !present {
		return err
	}

	if key != "" {
		_, getErr := secrets.Get(secrets.APIKey)
		if errors.Is(getErr, secrets.ErrNotFound) {
			if err := secrets.Set(secrets.APIKey, key); err != nil {
				return fmt.Errorf("failed to move API key into the OS keyring: %w", err)
			}
		} else if getErr != nil {
			return getErr
		}
	}

	return config.StripLegacyAPIKey()
}
