package guacamole

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
	resource := Provider().ResourcesMap["guacamole_connection_ssh"]
	parameters := resource.Schema["parameters"].Elem.(*schema.Resource)
	fontSize := parameters.Schema["font_size"]
	if fontSize.ValidateDiagFunc == nil {
		t.Fatal("parameters.font_size must validate during planning")
	}

	if diags := fontSize.ValidateDiagFunc("12", nil); len(diags) != 0 {
		t.Errorf("supported font size 12 was rejected: %#v", diags)
	}
	if diags := fontSize.ValidateDiagFunc("13", nil); !diags.HasError() {
		t.Error("unsupported font size 13 was accepted")
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
