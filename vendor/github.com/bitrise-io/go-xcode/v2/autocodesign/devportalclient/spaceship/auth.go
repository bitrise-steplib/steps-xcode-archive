package spaceship

import (
	"encoding/json"
	"fmt"
)

// AuthClient ...
type AuthClient struct {
	client *Client
}

// NewAuthClient ...
func NewAuthClient(client *Client) *AuthClient {
	return &AuthClient{client: client}
}

// Login ...
func (c *AuthClient) Login() error {
	output, err := c.client.runSpaceshipCommand("login")
	if err != nil {
		return fmt.Errorf("running command failed with error: %w", err)
	}

	var teamIDResponse struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &teamIDResponse); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.client.teamID = teamIDResponse.Data
	return nil
}
