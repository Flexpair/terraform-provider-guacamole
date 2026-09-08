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

func TestCredentialSchemaFieldsAreSensitive(t *testing.T) {
	provider := Provider()

	assertSensitive := func(resource *schema.Resource, fields ...string) {
		t.Helper()
		parameters := resource.Schema["parameters"].Elem.(*schema.Resource).Schema
		for _, field := range fields {
			if !parameters[field].Sensitive {
				t.Errorf("parameters.%s must be marked Sensitive", field)
			}
		}
	}

	assertDataSourceSensitive := func(resource *schema.Resource, fields ...string) {
		t.Helper()
		parameters := resource.Schema["parameters"].Elem.(*schema.Resource).Schema
		for _, field := range fields {
			if !parameters[field].Sensitive {
				t.Errorf("data source parameters.%s must be marked Sensitive", field)
			}
		}
	}

	assertSensitive(provider.ResourcesMap["guacamole_connection_ssh"], "password", "private_key", "passphrase")
	assertSensitive(provider.ResourcesMap["guacamole_connection_vnc"], "password", "sftp_password", "sftp_private_key", "sftp_passphrase")
	assertSensitive(provider.ResourcesMap["guacamole_connection_rdp"], "password", "gateway_password", "sftp_password", "sftp_private_key", "sftp_passphrase")
	assertSensitive(provider.ResourcesMap["guacamole_connection_telnet"], "password")
	assertSensitive(provider.ResourcesMap["guacamole_connection_kubernetes"], "client_cert", "client_key")

	assertDataSourceSensitive(provider.DataSourcesMap["guacamole_connection_ssh"], "private_key", "passphrase")
	assertDataSourceSensitive(provider.DataSourcesMap["guacamole_connection_vnc"], "password", "sftp_password", "sftp_private_key", "sftp_passphrase")
	assertDataSourceSensitive(provider.DataSourcesMap["guacamole_connection_rdp"], "password", "gateway_password", "sftp_password", "sftp_private_key", "sftp_passphrase")
	assertDataSourceSensitive(provider.DataSourcesMap["guacamole_connection_kubernetes"], "client_cert", "client_key")
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
