package shared

import (
	"fmt"

	"github.com/lroolle/atlas-cli/api"
)

func GetConfluenceClient() (*api.ConfluenceClient, error) {
	client, err := api.GetConfluenceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Confluence client: %w (hint: set credentials in config file under 'confluence' section)", err)
	}
	return client, nil
}
