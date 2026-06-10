package guacamole

import (
	"errors"
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
