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

func TestConnectionCredentialFieldsAreSensitive(t *testing.T) {
	resourceCredentialFields := map[string][]string{
		"guacamole_connection_ssh":        {"password", "private_key", "passphrase"},
		"guacamole_connection_telnet":     {},
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

	for _, schemas := range []struct {
		kind   string
		fields map[string][]string
		items  map[string]*schema.Resource
	}{
		{kind: "resource", fields: resourceCredentialFields, items: Provider().ResourcesMap},
		{kind: "data source", fields: dataSourceCredentialFields, items: Provider().DataSourcesMap},
	} {
		for resourceName, fields := range schemas.fields {
			resource, ok := schemas.items[resourceName]
			if !ok {
				t.Errorf("%s %s is not registered", schemas.kind, resourceName)
				continue
			}
			parametersResource, ok := resource.Schema["parameters"].Elem.(*schema.Resource)
			if !ok {
				t.Errorf("%s %s parameters must be a nested resource", schemas.kind, resourceName)
				continue
			}
			for _, field := range fields {
				parameter, ok := parametersResource.Schema[field]
				if !ok {
					t.Errorf("%s %s parameters.%s is missing", schemas.kind, resourceName, field)
					continue
				}
				if !parameter.Sensitive {
					t.Errorf("%s %s parameters.%s must be marked Sensitive", schemas.kind, resourceName, field)
				}
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
