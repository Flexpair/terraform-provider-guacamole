package guacamole

import (
	"fmt"
	"log"
	"sync"
	"time"

	guac "github.com/techBeck03/guacamole-api-client"
)

// LazyClient defers Guacamole authentication until the first API call.
// This allows Terraform to plan resources even when the Guacamole server
// is not yet available (e.g., during HCP Terraform Stacks planning phase
// where the gateway VM has not been created yet).
//
// On first Get() call, it retries the connection with exponential backoff
// (up to ~10 minutes total) to handle the case where the gateway VM has
// just been launched and Docker/Guacamole is still starting up.
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
// Retries with exponential backoff if the server is not yet reachable.
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

	// Retry with exponential backoff: 15s, 30s, 60s, 60s, 60s, 60s, 60s, 60s, 60s, 60s
	// Total wait: ~9 minutes, giving the gateway VM time to boot
	backoff := 15 * time.Second
	maxBackoff := 60 * time.Second
	maxAttempts := 10

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client := guac.New(lc.config)
		if err := client.Connect(); err != nil {
			lastErr = err
			if attempt < maxAttempts {
				log.Printf("[INFO] Guacamole connection attempt %d/%d failed: %v. Retrying in %s...", attempt, maxAttempts, err, backoff)
				time.Sleep(backoff)
				if backoff < maxBackoff {
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
				continue
			}
		} else {
			if attempt > 1 {
				log.Printf("[INFO] Guacamole connection succeeded on attempt %d/%d", attempt, maxAttempts)
			}
			lc.client = &client
			return lc.client, nil
		}
	}

	lc.err = fmt.Errorf("unable to create guacamole client after %d attempts: %w", maxAttempts, lastErr)
	return nil, lc.err
}
