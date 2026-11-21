package guacamole

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	guac "github.com/techBeck03/guacamole-api-client"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"guacamole": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func TestConnectionFontSizeIsValidatedBySchema(t *testing.T) {
	provider := Provider()
	resources := []*schema.Resource{
		provider.ResourcesMap["guacamole_connection_ssh"],
		provider.ResourcesMap["guacamole_connection_telnet"],
		provider.ResourcesMap["guacamole_connection_kubernetes"],
	}
	for _, resource := range resources {
		parameters := resource.Schema["parameters"].Elem.(*schema.Resource)
		fontSize := parameters.Schema["font_size"]
		if fontSize.ValidateDiagFunc == nil {
			t.Error("parameters.font_size must validate during planning")
			continue
		}

		if diags := fontSize.ValidateDiagFunc("", nil); len(diags) != 0 {
			t.Errorf("rejected an empty font size: %#v", diags)
		}
		if diags := fontSize.ValidateDiagFunc("12", nil); len(diags) != 0 {
			t.Errorf("rejected supported font size 12: %#v", diags)
		}
		if diags := fontSize.ValidateDiagFunc("13", nil); !diags.HasError() {
			t.Error("accepted unsupported font size 13")
		}
	}
}

func TestConnectionPlanTimeValidationAttached(t *testing.T) {
	provider := Provider()
	cases := []struct {
		resourceName string
		parameter    string
	}{
		{resourceName: "guacamole_connection_rdp", parameter: "security_mode"},
		{resourceName: "guacamole_connection_ssh", parameter: "color_scheme"},
		{resourceName: "guacamole_connection_vnc", parameter: "cursor"},
		{resourceName: "guacamole_connection_telnet", parameter: "terminal_type"},
		{resourceName: "guacamole_connection_kubernetes", parameter: "backspace"},
		{resourceName: "guacamole_connection_ssh", parameter: "port"},
		{resourceName: "guacamole_connection_rdp", parameter: "timezone"},
	}
	for _, tc := range cases {
		resource, ok := provider.ResourcesMap[tc.resourceName]
		if !ok {
			t.Errorf("resource %s is not registered", tc.resourceName)
			continue
		}
		parameters, ok := resource.Schema["parameters"].Elem.(*schema.Resource)
		if !ok {
			t.Errorf("resource %s parameters must be a nested resource", tc.resourceName)
			continue
		}
		field, ok := parameters.Schema[tc.parameter]
		if !ok {
			t.Errorf("resource %s parameters.%s is missing", tc.resourceName, tc.parameter)
			continue
		}
		if field.ValidateDiagFunc == nil {
			t.Errorf("resource %s parameters.%s must validate during planning", tc.resourceName, tc.parameter)
		}
	}

	// Spot-check behavior: restricted values and integers reject bad input at
	// plan time but allow empty values (Guacamole applies its own defaults).
	rdp := provider.ResourcesMap["guacamole_connection_rdp"]
	securityMode := rdp.Schema["parameters"].Elem.(*schema.Resource).Schema["security_mode"]
	if diags := securityMode.ValidateDiagFunc("bogus", nil); !diags.HasError() {
		t.Error("accepted unsupported security_mode at plan time")
	}
	if diags := securityMode.ValidateDiagFunc("", nil); len(diags) != 0 {
		t.Errorf("rejected an empty security_mode: %#v", diags)
	}
	if diags := securityMode.ValidateDiagFunc("nla", nil); len(diags) != 0 {
		t.Errorf("rejected supported security_mode nla: %#v", diags)
	}
	port := rdp.Schema["parameters"].Elem.(*schema.Resource).Schema["port"]
	if diags := port.ValidateDiagFunc("not-a-port", nil); !diags.HasError() {
		t.Error("accepted a non-integer port at plan time")
	}
	if diags := port.ValidateDiagFunc("3389", nil); len(diags) != 0 {
		t.Errorf("rejected integer port 3389: %#v", diags)
	}
	timezone := rdp.Schema["parameters"].Elem.(*schema.Resource).Schema["timezone"]
	if diags := timezone.ValidateDiagFunc("Not/A_Zone", nil); !diags.HasError() {
		t.Error("accepted an invalid timezone at plan time")
	}
	if diags := timezone.ValidateDiagFunc("America/Chicago", nil); len(diags) != 0 {
		t.Errorf("rejected valid timezone America/Chicago: %#v", diags)
	}

	// Connection group: type enumeration and integer-like attributes.
	group := provider.ResourcesMap["guacamole_connection_group"]
	typeField := group.Schema["type"]
	if diags := typeField.ValidateDiagFunc("nonsense", nil); !diags.HasError() {
		t.Error("accepted an unsupported connection group type at plan time")
	}
	if diags := typeField.ValidateDiagFunc("organizational", nil); len(diags) != 0 {
		t.Errorf("rejected lowercase organizational (StateFunc uppercases): %#v", diags)
	}
}

func TestConnectionCredentialFieldsAreSensitive(t *testing.T) {
	resourceCredentialFields := map[string][]string{
		"guacamole_connection_ssh":        {"password", "private_key", "passphrase"},
		"guacamole_connection_telnet":     {"password"},
		"guacamole_connection_rdp":        {"password", "gateway_password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
		"guacamole_connection_vnc":        {"password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
		"guacamole_connection_kubernetes": {"client_cert", "client_key"},
	}
	dataSourceCredentialFields := map[string][]string{
		"guacamole_connection_ssh":        {"private_key", "passphrase"},
		"guacamole_connection_rdp":        {"password", "gateway_password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
		"guacamole_connection_vnc":        {"password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
		"guacamole_connection_kubernetes": {"client_cert", "client_key"},
	}

	assertSensitiveFields(t, "resource", Provider().ResourcesMap, resourceCredentialFields)
	assertSensitiveFields(t, "data source", Provider().DataSourcesMap, dataSourceCredentialFields)
}

func assertSensitiveFields(t *testing.T, kind string, resources map[string]*schema.Resource, expected map[string][]string) {
	t.Helper()
	for resourceName, fields := range expected {
		resource, ok := resources[resourceName]
		if !ok {
			t.Errorf("%s %s is not registered", kind, resourceName)
			continue
		}
		parametersResource, ok := resource.Schema["parameters"].Elem.(*schema.Resource)
		if !ok {
			t.Errorf("%s %s parameters must be a nested resource", kind, resourceName)
			continue
		}
		for _, field := range fields {
			parameter, ok := parametersResource.Schema[field]
			if !ok {
				t.Errorf("%s %s parameters.%s is missing", kind, resourceName, field)
				continue
			}
			if !parameter.Sensitive {
				t.Errorf("%s %s parameters.%s must be marked Sensitive", kind, resourceName, field)
			}
		}
	}
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("GUACAMOLE_URL") == "" {
		t.Fatal("GUACAMOLE_URL must be set for acceptance tests")
	}
	if os.Getenv("GUACAMOLE_PASSWORD") == "" {
		if os.Getenv("GUACAMOLE_TOKEN") == "" {
			t.Fatal("GUACAMOLE_PASSWORD or GUACAMOLE_TOKEN must be set for acceptance tests")
		}
	} else {
		if os.Getenv("GUACAMOLE_USERNAME") == "" {
			t.Fatal("GUACAMOLE_USERNAME must be set for acceptance tests when GUACAMOLE_PASSWORD is set")
		}
	}
	if os.Getenv("GUACAMOLE_TOKEN") != "" {
		if os.Getenv("GUACAMOLE_DATA_SOURCE") == "" {
			t.Fatal("GUACAMOLE_DATA_SOURCE must be set for acceptance tests when GUACAMOLE_TOKEN is provided")
		}
	}
}

// acceptanceClient returns an authenticated API client for acceptance-test
// CheckDestroy callbacks. Provider meta is a *LazyClient since lazy init was
// introduced, so destroy checks must go through LazyClient.Get() instead of
// asserting *guac.Client directly (which panics).
func acceptanceClient() (*guac.Client, error) {
	lc, ok := testAccProvider.Meta().(*LazyClient)
	if !ok {
		return nil, fmt.Errorf("provider meta is %T, want *LazyClient", testAccProvider.Meta())
	}
	return lc.Get()
}

var connectionResourceTypes = map[string]bool{
	"guacamole_connection_ssh":        true,
	"guacamole_connection_rdp":        true,
	"guacamole_connection_vnc":        true,
	"guacamole_connection_telnet":     true,
	"guacamole_connection_kubernetes": true,
}

// testAccCheckGuacamoleConnectionDestroy verifies that no connection or
// connection group created during an acceptance test remains on the server.
func testAccCheckGuacamoleConnectionDestroy(s *terraform.State) error {
	client, err := acceptanceClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		switch {
		case rs.Type == "guacamole_connection_group":
			if _, err := client.ReadConnectionGroup(rs.Primary.ID); err == nil {
				return fmt.Errorf("connection group %q still exists", rs.Primary.ID)
			}
		case connectionResourceTypes[rs.Type]:
			if _, err := client.ReadConnection(rs.Primary.ID); err == nil {
				return fmt.Errorf("connection %q still exists", rs.Primary.ID)
			}
		}
	}
	return nil
}
