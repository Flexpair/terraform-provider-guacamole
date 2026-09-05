package guacamole

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestStringBoolConversions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  bool
	}{
		{name: "true", input: "true", want: true},
		{name: "false", input: "false", want: false},
		{name: "empty", input: "", want: false},
		{name: "invalid", input: "not-a-bool", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stringToBool(tc.input); got != tc.want {
				t.Fatalf("stringToBool(%q) = %t, want %t", tc.input, got, tc.want)
			}
		})
	}

	if got := boolToString(true); got != "true" {
		t.Fatalf("boolToString(true) = %q, want %q", got, "true")
	}
	if got := boolToString(false); got != "" {
		t.Fatalf("boolToString(false) = %q, want empty string", got)
	}
}

func TestSliceDiff(t *testing.T) {
	if got := sliceDiff([]string{"a", "b"}, []string{"b", "c"}, false); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("one-way sliceDiff = %#v, want %#v", got, []string{"a"})
	}
	if got := sliceDiff([]string{"a", "b"}, []string{"b", "c"}, true); !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatalf("two-way sliceDiff = %#v, want %#v", got, []string{"a", "c"})
	}
}

func TestStringInSlice(t *testing.T) {
	if got := stringInSlice([]string{"ssh", "rdp"}, []string{"ssh"}); len(got) != 0 {
		t.Fatalf("valid stringInSlice returned diagnostics: %#v", got)
	}

	got := stringInSlice([]string{"ssh", "rdp"}, []string{"vnc"})
	if len(got) != 1 {
		t.Fatalf("invalid stringInSlice returned %d diagnostics, want 1", len(got))
	}
	assertDiagnostic(t, got[0], diag.Error, "Invalid value entered", "vnc is not one of supported values: ssh, rdp")
}

func TestValidateStringFields(t *testing.T) {
	tests := []struct {
		name        string
		values      []interface{}
		integerKeys []string
		restricted  map[string][]string
		fieldKind   string
		want        int
	}{
		{
			name:        "empty block",
			values:      []interface{}{},
			integerKeys: []string{"port"},
			restricted:  map[string][]string{"mode": {"valid"}},
			fieldKind:   "parameter",
		},
		{
			name: "valid integer and restricted value",
			values: []interface{}{map[string]interface{}{
				"port": "22",
				"mode": "valid",
			}},
			integerKeys: []string{"port"},
			restricted:  map[string][]string{"mode": {"valid"}},
			fieldKind:   "parameter",
		},
		{
			name: "invalid integer",
			values: []interface{}{map[string]interface{}{
				"port": "not-an-integer",
			}},
			integerKeys: []string{"port"},
			fieldKind:   "parameter",
			want:        1,
		},
		{
			name: "invalid restricted value",
			values: []interface{}{map[string]interface{}{
				"mode": "invalid",
			}},
			restricted: map[string][]string{"mode": {"valid"}},
			fieldKind:  "parameter",
			want:       1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateStringFields(tc.values, tc.integerKeys, tc.restricted, tc.fieldKind)
			if len(got) != tc.want {
				t.Fatalf("validateStringFields() returned %d diagnostics, want %d: %#v", len(got), tc.want, got)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	tests := []struct {
		name   string
		values []interface{}
		want   int
	}{
		{name: "empty block", values: []interface{}{}},
		{name: "valid timezone", values: []interface{}{map[string]interface{}{"timezone": "UTC"}}},
		{name: "invalid timezone", values: []interface{}{map[string]interface{}{"timezone": "not/a-timezone"}}, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateTimezone(tc.values, "timezone")
			if len(got) != tc.want {
				t.Fatalf("validateTimezone() returned %d diagnostics, want %d: %#v", len(got), tc.want, got)
			}
		})
	}
}

func TestCheckForDuplicates(t *testing.T) {
	if got := checkForDuplicates([]string{"ssh", "rdp"}); len(got) != 0 {
		t.Fatalf("unique values returned diagnostics: %#v", got)
	}

	got := checkForDuplicates([]string{"ssh", "ssh", "rdp", "rdp"})
	if len(got) != 1 {
		t.Fatalf("duplicate values returned %d diagnostics, want 1", len(got))
	}
	assertDiagnostic(t, got[0], diag.Error, "Duplicate entries found in array", "Found the duplicate entries: ssh, rdp")
}

func TestSortSliceBySlice(t *testing.T) {
	got := sortSliceBySlice([]string{"rdp", "ssh"}, []string{"ssh", "vnc", "rdp"})
	want := []string{"rdp", "ssh", "vnc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortSliceBySlice() = %#v, want %#v", got, want)
	}
}

func TestHCLConversion(t *testing.T) {
	if got := toHclString([]string{"ssh", "rdp"}, false); got != `["ssh", "rdp"]` {
		t.Fatalf("slice HCL = %q, want %q", got, `["ssh", "rdp"]`)
	}
	if got := toHclString(map[string]interface{}{"username": "jens", "enabled": true}, false); got != "{\nusername = \"jens\"\nenabled = true\n}" && got != "{\nenabled = true\nusername = \"jens\"\n}" {
		t.Fatalf("map HCL = %q, want both entries", got)
	}
	if got := toHclString(nil, false); got != "null" {
		t.Fatalf("nil HCL = %q, want %q", got, "null")
	}
	if got := primitiveToHclString("jens", true); got != `"jens"` {
		t.Fatalf("nested string HCL = %q, want %q", got, `"jens"`)
	}
}

func TestGenericConversions(t *testing.T) {
	if _, ok := tryToConvertToGenericSlice(nil); ok {
		t.Fatal("nil converted to generic slice")
	}
	if _, ok := tryToConvertToGenericMap(nil); ok {
		t.Fatal("nil converted to generic map")
	}
	if got, ok := tryToConvertToGenericSlice([]string{"ssh"}); !ok || !reflect.DeepEqual(got, []interface{}{"ssh"}) {
		t.Fatalf("slice conversion = %#v, %t", got, ok)
	}
	if _, ok := tryToConvertToGenericSlice("ssh"); ok {
		t.Fatal("string converted to generic slice")
	}
	if got, ok := tryToConvertToGenericMap(map[string]int{"port": 22}); !ok || got["port"] != 22 {
		t.Fatalf("map conversion = %#v, %t", got, ok)
	}
	if _, ok := tryToConvertToGenericMap(map[int]string{22: "ssh"}); ok {
		t.Fatal("non-string-key map converted to generic map")
	}
}

func TestValidateTimestring(t *testing.T) {
	if got := validateTimestring("2026-09-05", "valid_from"); len(got) != 0 {
		t.Fatalf("valid date returned diagnostics: %#v", got)
	}

	got := validateTimestring("05/09/2026", "valid_from")
	if len(got) != 1 {
		t.Fatalf("invalid date returned %d diagnostics, want 1", len(got))
	}
	assertDiagnostic(t, got[0], diag.Error, "Invalid timestring format for: valid_from", "Date string must be in the form of YYYY-MM-DD")
}

func TestValidateUser(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]interface{}
		want       int
	}{
		{
			name: "invalid timezone and dates",
			attributes: map[string]interface{}{
				"timezone":    "not/a-timezone",
				"valid_from":  "05/09/2026",
				"valid_until": "05/10/2026",
			},
			want: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, guacamoleUser().Schema, map[string]interface{}{
				"username":   "test-user",
				"attributes": []interface{}{tc.attributes},
			})
			if got := len(validateUser(d)); got != tc.want {
				t.Fatalf("validateUser() returned %d diagnostics, want %d", got, tc.want)
			}
		})
	}
}

func TestResourceSchemas(t *testing.T) {
	for _, tc := range []struct {
		name   string
		res    func() *schema.Resource
		fields []string
	}{
		{name: "user", res: guacamoleUser, fields: []string{"username", "attributes", "group_membership"}},
		{name: "user group", res: guacamoleUserGroup, fields: []string{"identifier", "attributes", "system_permissions"}},
		{name: "connection group", res: guacamoleConnectionGroup, fields: []string{"name", "attributes"}},
		{name: "ssh connection", res: guacamoleConnectionSSH, fields: []string{"name", "parameters"}},
		{name: "rdp connection", res: guacamoleConnectionRDP, fields: []string{"name", "parameters"}},
		{name: "vnc connection", res: guacamoleConnectionVNC, fields: []string{"name", "parameters"}},
		{name: "telnet connection", res: guacamoleConnectionTelnet, fields: []string{"name", "parameters"}},
		{name: "kubernetes connection", res: guacamoleConnectionKubernetes, fields: []string{"name", "parameters"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := tc.res()
			if resource == nil || resource.Schema == nil {
				t.Fatal("resource schema is nil")
			}
			for _, field := range tc.fields {
				if resource.Schema[field] == nil {
					t.Fatalf("schema field %q is missing", field)
				}
			}
		})
	}
}

func TestTestAccCheckTestSliceVals(t *testing.T) {
	state := &terraform.State{Modules: []*terraform.ModuleState{{
		Path: []string{"root"},
		Resources: map[string]*terraform.ResourceState{
			"guacamole_user.new": {
				Primary: &terraform.InstanceState{Attributes: map[string]string{
					"system_permissions.#": "2",
					"system_permissions.0": "READ",
					"system_permissions.1": "UPDATE",
				}},
			},
		},
	}}}

	if err := testAccCheckTestSliceVals("guacamole_user.new", "system_permissions", []string{"UPDATE", "READ"})(state); err != nil {
		t.Fatalf("matching state returned error: %v", err)
	}
	if err := testAccCheckTestSliceVals("guacamole_user.new", "system_permissions", []string{"READ"})(state); err == nil {
		t.Fatal("mismatching state returned nil error")
	}
	if err := testAccCheckTestSliceVals("missing", "system_permissions", []string{"READ"})(state); err == nil {
		t.Fatal("missing resource returned nil error")
	}
}

func assertDiagnostic(t *testing.T, got diag.Diagnostic, wantSeverity diag.Severity, wantSummary, wantDetail string) {
	t.Helper()
	if got.Severity != wantSeverity || got.Summary != wantSummary || got.Detail != wantDetail {
		t.Fatalf("diagnostic = %#v, want severity=%v summary=%q detail=%q", got, wantSeverity, wantSummary, wantDetail)
	}
}
