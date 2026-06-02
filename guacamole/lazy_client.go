package guacamole

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	guac "github.com/techBeck03/guacamole-api-client"
)

// LazyClient defers Guacamole authentication until the first API call.
// This allows Terraform to plan resources even when the Guacamole server
// is not yet available (e.g., during HCP Terraform Stacks planning phase
// where the gateway VM has not been created yet).
//
// On first Get() call, it polls the connection every 15 seconds for up to
// roughly 30 minutes to handle the case where the gateway VM has just been
// launched and Docker/Guacamole is still starting up.
//
// When the provider URL is empty (common during Stacks multi-wave planning),
// the client operates in "unconfigured" mode: Get() returns nil without error,
// and resource Read functions treat this as "resource not found".
type LazyClient struct {
	config guac.Config
	client *guac.Client
	mu     sync.Mutex
	err    error

	connect       func(guac.Config) (*guac.Client, error)
	sleep         func(time.Duration)
	retryInterval time.Duration
	maxAttempts   int
}

const (
	defaultRetryInterval = 15 * time.Second
	defaultMaxAttempts   = 120
)

// NewLazyClient stores the provider configuration without connecting.
func NewLazyClient(config guac.Config) *LazyClient {
	return &LazyClient{
		config:        config,
		connect:       connectGuacamole,
		sleep:         time.Sleep,
		retryInterval: defaultRetryInterval,
		maxAttempts:   defaultMaxAttempts,
	}
}

func connectGuacamole(config guac.Config) (*guac.Client, error) {
	client := guac.New(config)
	if err := client.Connect(); err != nil {
		return nil, err
	}
	return &client, nil
}

// IsConfigured reports whether the provider has a non-empty server URL.
func (lc *LazyClient) IsConfigured() bool {
	return lc.config.URL != ""
}

// readWithClient returns an authenticated client for Read operations.
// If the provider URL is empty (Stacks deferred planning), it clears the
// resource ID and returns nil — telling Terraform the resource doesn't exist
// yet, so the plan shows "create" instead of erroring.
func readWithClient(m interface{}, d *schema.ResourceData) (*guac.Client, diag.Diagnostics) {
	lc := m.(*LazyClient)
	if !lc.IsConfigured() {
		d.SetId("")
		return nil, nil
	}
	client, err := lc.Get()
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return client, nil
}

// writeWithClient returns an authenticated client for Create/Update/Delete.
// If the provider URL is empty, it returns an error — write operations
// cannot proceed without a configured server.
func writeWithClient(m interface{}) (*guac.Client, diag.Diagnostics) {
	lc := m.(*LazyClient)
	if !lc.IsConfigured() {
		return nil, diag.Errorf("guacamole provider: server URL not configured (gateway not yet deployed)")
	}
	client, err := lc.Get()
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return client, nil
}

// Get returns an authenticated Guacamole client, connecting on first call.
// Polls at a fixed interval if the server is not yet reachable.
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

	connect := lc.connect
	if connect == nil {
		connect = connectGuacamole
	}
	sleep := lc.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	retryInterval := lc.retryInterval
	if retryInterval < 0 {
		retryInterval = 0
	}
	maxAttempts := lc.maxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client, err := connect(lc.config)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts {
				log.Printf("[INFO] Guacamole connection attempt %d/%d failed: %v. Retrying in %s...", attempt, maxAttempts, err, retryInterval)
				sleep(retryInterval)
				continue
			}
		} else {
			if attempt > 1 {
				log.Printf("[INFO] Guacamole connection succeeded on attempt %d/%d", attempt, maxAttempts)
			}
			lc.client = client
			return lc.client, nil
		}
	}

	lc.err = fmt.Errorf("unable to create guacamole client after %d attempts: %w", maxAttempts, lastErr)
	return nil, lc.err
}
