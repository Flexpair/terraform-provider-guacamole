package guacamole

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	guac "github.com/techBeck03/guacamole-api-client"
)

// newLifecycleTestServer returns a fake Guacamole API. Token requests always
// succeed; every other request is answered with the provided status code so
// Read/Delete error handling can be exercised without a live server.
func newLifecycleTestServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/tokens" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"authToken":"fixture","dataSource":"test"}`))
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/userGroups") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"identifier": "test-group"})
			return
		}
		w.WriteHeader(status)
	}))
}

// TestReadWithClientUnconfiguredPreservesID covers the empty-URL mode used by
// HCP Terraform Stacks multi-wave planning: Read must leave existing state
// unchanged because an unavailable URL does not prove the resource is missing.
func TestReadWithClientUnconfiguredPreservesID(t *testing.T) {
	lc := NewLazyClient(guac.Config{})
	data := schema.TestResourceDataRaw(t, guacamoleUser().Schema, map[string]interface{}{})
	data.SetId("someone")

	client, diags := readWithClient(lc, data)
	if client != nil {
		t.Fatal("readWithClient returned a client for an unconfigured provider")
	}
	if diags.HasError() {
		t.Fatalf("readWithClient returned diagnostics: %#v", diags)
	}
	if data.Id() != "someone" {
		t.Fatalf("id = %q, want existing id preserved", data.Id())
	}
}

// TestWriteWithClientUnconfiguredErrors covers the empty-URL mode for
// Create/Update/Delete: writes must fail with a clear diagnostic instead of
// retrying against an empty URL.
func TestWriteWithClientUnconfiguredErrors(t *testing.T) {
	lc := NewLazyClient(guac.Config{})

	client, diags := writeWithClient(lc)
	if client != nil {
		t.Fatal("writeWithClient returned a client for an unconfigured provider")
	}
	if !diags.HasError() {
		t.Fatal("writeWithClient returned no error for an unconfigured provider")
	}
	if !strings.Contains(diags[0].Summary, "server URL not configured") {
		t.Fatalf("diagnostic summary = %q, want mention of missing server URL", diags[0].Summary)
	}
}

// TestResourceUserReadTreatsNotFoundAsMissing verifies that a replaced gateway
// (fresh database, stale state IDs) makes Read clear the ID instead of
// failing the apply.
func TestResourceUserReadTreatsNotFoundAsMissing(t *testing.T) {
	server := newLifecycleTestServer(t, http.StatusNotFound)
	defer server.Close()

	data := schema.TestResourceDataRaw(t, guacamoleUser().Schema, map[string]interface{}{"username": "ghost"})
	data.SetId("ghost")

	diags := resourceUserRead(context.Background(), data, NewLazyClient(guac.Config{URL: server.URL}))
	if diags.HasError() {
		t.Fatalf("resourceUserRead returned diagnostics for a missing user: %#v", diags)
	}
	if data.Id() != "" {
		t.Fatalf("id = %q, want cleared id so Terraform plans a recreate", data.Id())
	}
}

// TestResourceUserReadPropagatesServerErrors verifies that only 404s are
// treated as "resource missing"; other failures remain hard errors.
func TestResourceUserReadPropagatesServerErrors(t *testing.T) {
	server := newLifecycleTestServer(t, http.StatusInternalServerError)
	defer server.Close()

	data := schema.TestResourceDataRaw(t, guacamoleUser().Schema, map[string]interface{}{"username": "someone"})
	data.SetId("someone")

	diags := resourceUserRead(context.Background(), data, NewLazyClient(guac.Config{URL: server.URL}))
	if !diags.HasError() {
		t.Fatal("resourceUserRead swallowed a server error")
	}
	if data.Id() != "someone" {
		t.Fatalf("id = %q, want the original id to be preserved on error", data.Id())
	}
}

// TestResourceUserDeleteTreatsNotFoundAsSuccess verifies delete idempotency
// after the gateway database has been replaced.
func TestResourceUserDeleteTreatsNotFoundAsSuccess(t *testing.T) {
	server := newLifecycleTestServer(t, http.StatusNotFound)
	defer server.Close()

	data := schema.TestResourceDataRaw(t, guacamoleUser().Schema, map[string]interface{}{"username": "ghost"})
	data.SetId("ghost")

	diags := resourceUserDelete(context.Background(), data, NewLazyClient(guac.Config{URL: server.URL}))
	if diags.HasError() {
		t.Fatalf("resourceUserDelete failed on an already-deleted user: %#v", diags)
	}
	if data.Id() != "" {
		t.Fatalf("id = %q, want cleared id", data.Id())
	}
}

// TestResourceUserGroupCreateSurfacesReadErrors is a regression test for
// Create discarding the diagnostics of its post-create Read: a failed refresh
// must fail the operation instead of reporting success with stale state.
func TestResourceUserGroupCreateSurfacesReadErrors(t *testing.T) {
	// POST (create) succeeds via newLifecycleTestServer; every GET afterwards
	// returns 500 because this server answers non-token, non-create requests
	// with http.StatusInternalServerError.
	server := newLifecycleTestServer(t, http.StatusInternalServerError)
	defer server.Close()

	data := schema.TestResourceDataRaw(t, guacamoleUserGroup().Schema, map[string]interface{}{"identifier": "test-group"})

	diags := resourceUserGroupCreate(context.Background(), data, NewLazyClient(guac.Config{URL: server.URL}))
	if !diags.HasError() {
		t.Fatal("resourceUserGroupCreate reported success although its post-create Read failed")
	}
}
