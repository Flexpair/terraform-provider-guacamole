package guacamole

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	guac "github.com/techBeck03/guacamole-api-client"
	types "github.com/techBeck03/guacamole-api-client/types"
)

func TestUserPermissionUpdateDeltas(t *testing.T) {
	tests := []struct {
		name   string
		old    []string
		new    []string
		add    []string
		remove []string
	}{
		{name: "add", old: []string{"existing"}, new: []string{"existing", "added"}, add: []string{"added"}},
		{name: "remove", old: []string{"removed", "existing"}, new: []string{"existing"}, remove: []string{"removed"}},
		{name: "unchanged", old: []string{"existing"}, new: []string{"existing"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			added := sliceDiff(tc.new, tc.old, false)
			removed := sliceDiff(tc.old, tc.new, false)
			if !reflect.DeepEqual(added, tc.add) {
				t.Fatalf("added delta = %#v, want %#v", added, tc.add)
			}
			if !reflect.DeepEqual(removed, tc.remove) {
				t.Fatalf("removed delta = %#v, want %#v", removed, tc.remove)
			}
		})
	}
}

func TestUserGroupPermissionUpdateDeltas(t *testing.T) {
	old := []string{"group-old", "group-shared"}
	new := []string{"group-shared", "group-new"}

	if got := sliceDiff(old, new, false); !reflect.DeepEqual(got, []string{"group-old"}) {
		t.Fatalf("removed group delta = %#v", got)
	}
	if got := sliceDiff(new, old, false); !reflect.DeepEqual(got, []string{"group-new"}) {
		t.Fatalf("added group delta = %#v", got)
	}
}

func TestUserPermissionSchemasUseSets(t *testing.T) {
	for _, resource := range []*schema.Resource{guacamoleUser(), guacamoleUserGroup()} {
		for _, field := range []string{"group_membership", "system_permissions", "connections", "connection_groups"} {
			if resource.Schema[field] == nil {
				continue
			}
			if resource.Schema[field].Type != schema.TypeSet {
				t.Fatalf("schema field %q has type %v, want TypeSet", field, resource.Schema[field].Type)
			}
		}
	}
}

func TestSystemPermissionChoices(t *testing.T) {
	choices := types.SystemPermissions{}.ValidChoices()
	if got := stringInSlice(choices, []string{"CREATE_USER", "CREATE_CONNECTION"}); got.HasError() {
		t.Fatalf("valid system permissions returned diagnostics: %#v", got)
	}
	if got := stringInSlice(choices, []string{"NOT_A_PERMISSION"}); !got.HasError() {
		t.Fatal("invalid system permission returned no diagnostics")
	}
}

func TestPermissionItemConstructors(t *testing.T) {
	client := guac.New(guac.Config{})
	tests := []struct {
		name string
		got  types.GuacPermissionItem
		want types.GuacPermissionItem
	}{
		{name: "add group member", got: client.NewAddGroupMemberPermission("ops"), want: types.GuacPermissionItem{Op: "add", Path: "/", Value: "ops"}},
		{name: "remove group member", got: client.NewRemoveGroupMemberPermission("ops"), want: types.GuacPermissionItem{Op: "remove", Path: "/", Value: "ops"}},
		{name: "add system permission", got: client.NewAddSystemPermission("CREATE_USER"), want: types.GuacPermissionItem{Op: "add", Path: "/systemPermissions", Value: "CREATE_USER"}},
		{name: "remove system permission", got: client.NewRemoveSystemPermission("CREATE_USER"), want: types.GuacPermissionItem{Op: "remove", Path: "/systemPermissions", Value: "CREATE_USER"}},
		{name: "add connection", got: client.NewAddConnectionPermission("42"), want: types.GuacPermissionItem{Op: "add", Path: "/connectionPermissions/42", Value: "READ"}},
		{name: "remove connection", got: client.NewRemoveConnectionPermission("42"), want: types.GuacPermissionItem{Op: "remove", Path: "/connectionPermissions/42", Value: "READ"}},
		{name: "add connection group", got: client.NewAddConnectionGroupPermission("servers"), want: types.GuacPermissionItem{Op: "add", Path: "/connectionGroupPermissions/servers", Value: "READ"}},
		{name: "remove connection group", got: client.NewRemoveConnectionGroupPermission("servers"), want: types.GuacPermissionItem{Op: "remove", Path: "/connectionGroupPermissions/servers", Value: "READ"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Fatalf("permission item = %#v, want %#v", tc.got, tc.want)
			}
		})
	}
}

type permissionUpdateCase struct {
	name  string
	field string
	old   []interface{}
	new   []interface{}
	want  []types.GuacPermissionItem
}

type permissionUpdateSuite struct {
	name     string
	resource *schema.Resource
	id       string
	wantPath string
	update   func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics
	cases    []permissionUpdateCase
}

func newPermissionUpdateCase(name, field string, old, new []interface{}, want ...types.GuacPermissionItem) permissionUpdateCase {
	return permissionUpdateCase{name: name, field: field, old: old, new: new, want: want}
}

func systemPermissionUpdateCase() permissionUpdateCase {
	return newPermissionUpdateCase("system permissions", "system_permissions", []interface{}{"CREATE_USER"}, []interface{}{"CREATE_CONNECTION"},
		types.GuacPermissionItem{Op: "remove", Path: "/systemPermissions", Value: "CREATE_USER"},
		types.GuacPermissionItem{Op: "add", Path: "/systemPermissions", Value: "CREATE_CONNECTION"})
}

func connectionPermissionUpdateCase() permissionUpdateCase {
	return newPermissionUpdateCase("connections", "connections", []interface{}{"old-connection"}, []interface{}{"new-connection"},
		types.GuacPermissionItem{Op: "remove", Path: "/connectionPermissions/old-connection", Value: "READ"},
		types.GuacPermissionItem{Op: "add", Path: "/connectionPermissions/new-connection", Value: "READ"})
}

func connectionGroupPermissionUpdateCase() permissionUpdateCase {
	return newPermissionUpdateCase("connection groups", "connection_groups", []interface{}{"old-group"}, []interface{}{"new-group"},
		types.GuacPermissionItem{Op: "remove", Path: "/connectionGroupPermissions/old-group", Value: "READ"},
		types.GuacPermissionItem{Op: "add", Path: "/connectionGroupPermissions/new-group", Value: "READ"})
}

func newPermissionUpdateSuite(name, id, path string, resource *schema.Resource, update func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics, cases ...permissionUpdateCase) permissionUpdateSuite {
	return permissionUpdateSuite{name: name, resource: resource, id: id, wantPath: path, update: update, cases: cases}
}

func TestResourcePermissionUpdatesSendDeltas(t *testing.T) {
	suites := []permissionUpdateSuite{
		newPermissionUpdateSuite("user", "test-user", "/api/session/data/test/users/test-user/permissions", guacamoleUser(), resourceUserUpdate,
			systemPermissionUpdateCase(), connectionGroupPermissionUpdateCase()),
		newPermissionUpdateSuite("user group", "test-group", "/api/session/data/test/userGroups/test-group/permissions", guacamoleUserGroup(), resourceUserGroupUpdate,
			systemPermissionUpdateCase(), connectionPermissionUpdateCase(), connectionGroupPermissionUpdateCase()),
	}

	for _, suite := range suites {
		t.Run(suite.name, func(t *testing.T) {
			for _, tc := range suite.cases {
				t.Run(tc.name, func(t *testing.T) {
					path, items, err := runPermissionUpdate(t, suite.resource, suite.id, tc.field, tc.old, tc.new, suite.update)
					if err != nil {
						t.Fatal(err)
					}
					if path != suite.wantPath {
						t.Fatalf("permission patch path = %q, want %q", path, suite.wantPath)
					}
					if !reflect.DeepEqual(items, tc.want) {
						t.Fatalf("permission patch = %#v, want %#v", items, tc.want)
					}
				})
			}
		})
	}
}

func TestResourceUserUpdateSendsConnectionPermissionDelta(t *testing.T) {
	var patchPath string
	var patchItems []types.GuacPermissionItem
	var patchErr error
	server := newPermissionTestServer(t, func(r *http.Request) {
		patchPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			patchErr = err
			return
		}
		patchErr = json.Unmarshal(body, &patchItems)
	})
	defer server.Close()

	lazyClient := newPermissionTestLazyClient(t, server.URL)
	data := changedResourceData(t, guacamoleUser(), "test-user",
		map[string]interface{}{"username": "test-user", "connections": []interface{}{"old"}},
		map[string]interface{}{"username": "test-user", "connections": []interface{}{"new"}},
	)

	if diags := resourceUserUpdate(context.Background(), data, lazyClient); diags.HasError() {
		t.Fatalf("resourceUserUpdate() returned diagnostics: %#v", diags)
	}
	if patchErr != nil {
		t.Fatalf("decode connection permission patch: %v", patchErr)
	}

	want := []types.GuacPermissionItem{
		{Op: "remove", Path: "/connectionPermissions/old", Value: "READ"},
		{Op: "add", Path: "/connectionPermissions/new", Value: "READ"},
	}
	if patchPath != "/api/session/data/test/users/test-user/permissions" {
		t.Fatalf("permission patch path = %q", patchPath)
	}
	if !reflect.DeepEqual(patchItems, want) {
		t.Fatalf("permission patch = %#v, want %#v", patchItems, want)
	}
}

func TestResourceUserGroupUpdateSendsMembershipDelta(t *testing.T) {
	var patchPath string
	var patchItems []types.GuacPermissionItem
	var patchErr error
	server := newPermissionTestServer(t, func(r *http.Request) {
		patchPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			patchErr = err
			return
		}
		patchErr = json.Unmarshal(body, &patchItems)
	})
	defer server.Close()

	lazyClient := newPermissionTestLazyClient(t, server.URL)
	data := changedResourceData(t, guacamoleUserGroup(), "test-group",
		map[string]interface{}{"identifier": "test-group"},
		map[string]interface{}{"identifier": "test-group", "group_membership": []interface{}{"parent"}},
	)

	if diags := resourceUserGroupUpdate(context.Background(), data, lazyClient); diags.HasError() {
		t.Fatalf("resourceUserGroupUpdate() returned diagnostics: %#v", diags)
	}
	if patchErr != nil {
		t.Fatalf("decode membership permission patch: %v", patchErr)
	}

	want := []types.GuacPermissionItem{{Op: "add", Path: "/", Value: "parent"}}
	if patchPath != "/api/session/data/test/userGroups/test-group/memberUserGroups" {
		t.Fatalf("membership patch path = %q", patchPath)
	}
	if !reflect.DeepEqual(patchItems, want) {
		t.Fatalf("membership patch = %#v, want %#v", patchItems, want)
	}
}

func runPermissionUpdate(t *testing.T, resource *schema.Resource, id, field string, old, new []interface{}, update func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics) (string, []types.GuacPermissionItem, error) {
	t.Helper()

	var patchPath string
	var patchItems []types.GuacPermissionItem
	var patchErr error
	server := newPermissionTestServer(t, func(r *http.Request) {
		patchPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			patchErr = err
			return
		}
		patchErr = json.Unmarshal(body, &patchItems)
	})
	defer server.Close()

	lazyClient := newPermissionTestLazyClient(t, server.URL)
	oldConfig := map[string]interface{}{}
	newConfig := map[string]interface{}{field: new}
	if resource.Schema["username"] != nil {
		oldConfig["username"] = id
		newConfig["username"] = id
	} else {
		oldConfig["identifier"] = id
		newConfig["identifier"] = id
	}
	oldConfig[field] = old
	data := changedResourceData(t, resource, id, oldConfig, newConfig)

	if diags := update(context.Background(), data, lazyClient); diags.HasError() {
		return patchPath, patchItems, fmt.Errorf("resource update returned diagnostics: %#v", diags)
	}
	if patchErr != nil {
		return patchPath, patchItems, fmt.Errorf("decode permission patch: %w", patchErr)
	}
	return patchPath, patchItems, nil
}

func newPermissionTestLazyClient(t *testing.T, serverURL string) *LazyClient {
	t.Helper()
	client := guac.New(guac.Config{URL: serverURL})
	if err := client.Connect(); err != nil {
		t.Fatalf("connect test client: %v", err)
	}

	lazyClient := NewLazyClient(guac.Config{URL: serverURL})
	lazyClient.connect = func(guac.Config) (*guac.Client, error) {
		return &client, nil
	}
	return lazyClient
}

func newPermissionTestServer(t *testing.T, patchHandler func(*http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/tokens" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"authToken":"fixture","dataSource":"test"}`))
			return
		}

		if r.Method == http.MethodPatch {
			patchHandler(r)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		switch r.URL.Path {
		case "/api/session/data/test/users/test-user":
			_ = json.NewEncoder(w).Encode(types.GuacUser{Username: "test-user", Attributes: types.GuacUserAttributes{Timezone: "UTC"}})
		case "/api/session/data/test/users/test-user/userGroups":
			_ = json.NewEncoder(w).Encode([]string{})
		case "/api/session/data/test/users/test-user/permissions":
			_ = json.NewEncoder(w).Encode(types.GuacPermissionData{})
		case "/api/session/data/test/userGroups":
			_ = json.NewEncoder(w).Encode(map[string]types.GuacUserGroup{"parent": {Identifier: "parent"}})
		case "/api/session/data/test/userGroups/test-group":
			_ = json.NewEncoder(w).Encode(types.GuacUserGroup{Identifier: "test-group"})
		case "/api/session/data/test/userGroups/test-group/memberUserGroups":
			_ = json.NewEncoder(w).Encode([]string{})
		case "/api/session/data/test/userGroups/test-group/permissions":
			_ = json.NewEncoder(w).Encode(types.GuacPermissionData{})
		default:
			http.NotFound(w, r)
		}
	}))
}

func changedResourceData(t *testing.T, resource *schema.Resource, id string, old, new map[string]interface{}) *schema.ResourceData {
	t.Helper()
	oldData := schema.TestResourceDataRaw(t, resource.Schema, old)
	oldData.SetId(id)
	state := oldData.State()
	if state == nil {
		t.Fatal("old resource state is nil")
	}

	diff, err := schema.InternalMap(resource.Schema).Diff(context.Background(), state, terraform.NewResourceConfigRaw(new), nil, nil, true)
	if err != nil {
		t.Fatalf("build resource diff: %v", err)
	}
	data, err := schema.InternalMap(resource.Schema).Data(state, diff)
	if err != nil {
		t.Fatalf("build changed resource data: %v", err)
	}
	return data
}
