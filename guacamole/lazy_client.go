package guacamole

import (
	"fmt"
	"sync"

	guac "github.com/techBeck03/guacamole-api-client"
)

// LazyClient defers Guacamole authentication until the first API call.
// This allows Terraform to plan resources even when the Guacamole server
// is not yet available (e.g., during HCP Terraform Stacks planning phase
// where the gateway VM has not been created yet).
type LazyClient struct {
	config guac.Config
	client *guac.Client
	mu     sync.Mutex
	err    error
}

// NewLazyClient stores the provider configuration without connecting.
func NewLazyClient(config guac.Config) *LazyClient {
	return &LazyClient{config: config}
}

// Get returns an authenticated Guacamole client, connecting on first call.
// Subsequent calls return the cached client. Thread-safe.
func (lc *LazyClient) Get() (*guac.Client, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.client != nil {
		return lc.client, nil
	}
	if lc.err != nil {
		return nil, lc.err
	}

	client := guac.New(lc.config)
	if err := client.Connect(); err != nil {
		lc.err = fmt.Errorf("unable to create guacamole client: %w", err)
		return nil, lc.err
	}
	lc.client = &client
	return lc.client, nil
}
