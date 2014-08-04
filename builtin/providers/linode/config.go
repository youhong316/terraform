package linode

import (
	"log"

	"github.com/pearkes/linode"
)

type Config struct {
	Key string `mapstructure:"key"`
}

// Client() returns a new client for accessing linode
//
func (c *Config) Client() (*linode.Client, error) {
	client, err := linode.NewClient(c.Key)

	log.Printf("[INFO] Linode Client configured for URL: %s", client.URL)

	if err != nil {
		return nil, err
	}

	return client, nil
}
