package guacamole

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	guac "github.com/techBeck03/guacamole-api-client"
)

func TestLazyClientDefaultRetryPolicy(t *testing.T) {
	client := NewLazyClient(guac.Config{URL: "https://example.test/guacamole"})

	if client.retryInterval != 15*time.Second {
		t.Fatalf("retryInterval = %s, want 15s", client.retryInterval)
	}
	if client.maxAttempts != 120 {
		t.Fatalf("maxAttempts = %d, want 120", client.maxAttempts)
	}
}

func TestLazyClientGetPollsUntilConnectSucceeds(t *testing.T) {
	lazyClient := NewLazyClient(guac.Config{URL: "https://example.test/guacamole"})
	lazyClient.maxAttempts = 4
	lazyClient.retryInterval = 15 * time.Second

	var attempts int
	var sleeps []time.Duration
	lazyClient.connect = func(config guac.Config) (*guac.Client, error) {
		attempts++
		if attempts < 4 {
			return nil, errors.New("guacamole not ready")
		}
		client := guac.New(config)
		return &client, nil
	}
	lazyClient.sleep = func(duration time.Duration) {
		sleeps = append(sleeps, duration)
	}

	client, err := lazyClient.Get()
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if client == nil {
		t.Fatal("Get() returned nil client")
	}
	if attempts != 4 {
		t.Fatalf("attempts = %d, want 4", attempts)
	}
	if len(sleeps) != 3 {
		t.Fatalf("sleep count = %d, want 3", len(sleeps))
	}
	for i, sleep := range sleeps {
		if sleep != 15*time.Second {
			t.Fatalf("sleep[%d] = %s, want 15s", i, sleep)
		}
	}

	_, err = lazyClient.Get()
	if err != nil {
		t.Fatalf("second Get() returned error: %v", err)
	}
	if attempts != 4 {
		t.Fatalf("second Get() attempted reconnect; attempts = %d, want 4", attempts)
	}
}

func TestLazyClientGetCachesFinalError(t *testing.T) {
	lazyClient := NewLazyClient(guac.Config{URL: "https://example.test/guacamole"})
	lazyClient.maxAttempts = 3
	lazyClient.retryInterval = 15 * time.Second

	var attempts int
	var sleeps []time.Duration
	lazyClient.connect = func(config guac.Config) (*guac.Client, error) {
		attempts++
		return nil, errors.New("still not ready")
	}
	lazyClient.sleep = func(duration time.Duration) {
		sleeps = append(sleeps, duration)
	}

	_, err := lazyClient.Get()
	if err == nil {
		t.Fatal("Get() returned nil error, want failure")
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(sleeps) != 2 {
		t.Fatalf("sleep count = %d, want 2", len(sleeps))
	}

	_, err = lazyClient.Get()
	if err == nil {
		t.Fatal("second Get() returned nil error, want cached failure")
	}
	if attempts != 3 {
		t.Fatalf("second Get() attempted reconnect; attempts = %d, want 3", attempts)
	}
}

func TestLazyClientDoesNotExposeConnectionErrors(t *testing.T) {
	lazyClient := NewLazyClient(guac.Config{URL: "https://example.test/guacamole"})
	lazyClient.maxAttempts = 2
	lazyClient.retryInterval = time.Second
	lazyClient.connect = func(guac.Config) (*guac.Client, error) {
		return nil, errors.New("request headers: Guacamole-Token: secret-token; Cookie: secret-cookie")
	}
	lazyClient.sleep = func(time.Duration) {}

	var logs bytes.Buffer
	originalLogOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(originalLogOutput)
	})

	_, err := lazyClient.Get()
	if err == nil {
		t.Fatal("Get() returned nil error, want failure")
	}
	for _, secret := range []string{"secret-token", "secret-cookie"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("Get() error exposed %q: %v", secret, err)
		}
		if strings.Contains(logs.String(), secret) {
			t.Errorf("connection logs exposed %q: %s", secret, logs.String())
		}
	}
	if !strings.Contains(err.Error(), "after 2 attempts") {
		t.Errorf("Get() error = %q, want attempt count", err)
	}
	if !strings.Contains(logs.String(), "connection attempt 1/2 failed") {
		t.Errorf("connection log = %q, want retry status", logs.String())
	}
}

func TestLazyClientRetryPolicyNormalizesInvalidValues(t *testing.T) {
	lazyClient := NewLazyClient(guac.Config{URL: "https://example.test/guacamole"})
	lazyClient.connect = nil
	lazyClient.sleep = nil
	lazyClient.retryInterval = -time.Second
	lazyClient.maxAttempts = 0

	connect, sleep, interval, attempts := lazyClient.retryPolicy()
	if connect == nil {
		t.Fatal("retryPolicy() returned nil connect function")
	}
	if sleep == nil {
		t.Fatal("retryPolicy() returned nil sleep function")
	}
	if interval != 0 {
		t.Fatalf("retry interval = %s, want 0", interval)
	}
	if attempts != 1 {
		t.Fatalf("max attempts = %d, want 1", attempts)
	}
}

func TestIsNotFoundError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{
			name: "404 from client Call",
			err:  errors.New("request &{...}\n failed with status code 404\n response map[]\n"),
			want: true,
		},
		{name: "other status code", err: errors.New("failed with status code 500"), want: false},
		{name: "unrelated error", err: errors.New("connection refused"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNotFoundError(tc.err); got != tc.want {
				t.Fatalf("isNotFoundError(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}
