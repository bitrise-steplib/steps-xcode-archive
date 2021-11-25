package appleauth

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/log"
)

// Inputs is Apple Service authentication configuration provided by end user
type Inputs struct {
	// Apple ID
	Username, Password, AppSpecificPassword string
	// API key (JWT)
	APIIssuer, APIKeyPath string
}

// Validate trims extra spaces and checks input grouping
func (cfg *Inputs) Validate() error {
	cfg.APIIssuer = strings.TrimSpace(cfg.APIIssuer)
	cfg.APIKeyPath = strings.TrimSpace(cfg.APIKeyPath)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.AppSpecificPassword = strings.TrimSpace(cfg.AppSpecificPassword)
	var (
		isAPIKeyAuthType  = (cfg.APIKeyPath != "" || cfg.APIIssuer != "")
		isAppleIDAuthType = (cfg.AppSpecificPassword != "" || cfg.Username != "" || cfg.Password != "")
	)

	switch {
	case isAppleIDAuthType && isAPIKeyAuthType:
		log.Warnf("Either provide Apple ID, Password (and  App-specific password if available) OR API Key Path and API Issuer")
		return fmt.Errorf("both Apple ID and API key related configuration provided, but only one of them expected")

	case isAppleIDAuthType:
		if cfg.AppSpecificPassword != "" {
			// App Specific Password provided, assuming 2FA is enabled.
			// In this case 2FA session is required, configured Bitrise account connection required, this contains username+password
			break
		}
		if cfg.Username == "" {
			return fmt.Errorf("no Apple Service Apple ID provided")
		}
		if cfg.Password == "" {
			return fmt.Errorf("no Apple Service Password provided")
		}

	case isAPIKeyAuthType:
		if cfg.APIIssuer == "" {
			return fmt.Errorf("no Apple Service API Issuer provided")
		}
		if cfg.APIKeyPath == "" {
			return fmt.Errorf("no Apple Service API Key Path provided")
		}
	}

	return nil
}
