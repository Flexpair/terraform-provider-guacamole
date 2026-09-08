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

	assertSensitive := func(kind string, resourceName string, resource *schema.Resource, fields []string) {
		t.Helper()
		if resource == nil {
			t.Fatalf("%s %s is not registered", kind, resourceName)
		}
		parameters := resource.Schema["parameters"].Elem.(*schema.Resource).Schema
		for _, field := range fields {
			schemaField, ok := parameters[field]
			if !ok {
				t.Errorf("%s %s has no parameters.%s field", kind, resourceName, field)
			} else if !schemaField.Sensitive {
				t.Errorf("%s %s parameters.%s must be marked Sensitive", kind, resourceName, field)
			}
		}
	}

	type credentialSchema struct {
		name           string
		resourceFields []string
		dataFields     []string
	}

	checks := []credentialSchema{
		{
			name:           "guacamole_connection_ssh",
			resourceFields: []string{"password", "private_key", "passphrase"},
			dataFields:     []string{"private_key", "passphrase"},
		},
		{
			name:           "guacamole_connection_vnc",
			resourceFields: []string{"password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
			dataFields:     []string{"password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
		},
		{
			name:           "guacamole_connection_rdp",
			resourceFields: []string{"password", "gateway_password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
			dataFields:     []string{"password", "gateway_password", "sftp_password", "sftp_private_key", "sftp_passphrase"},
		},
		{
			name:           "guacamole_connection_telnet",
			resourceFields: []string{"password"},
		},
		{
			name:           "guacamole_connection_kubernetes",
			resourceFields: []string{"client_cert", "client_key"},
			dataFields:     []string{"client_cert", "client_key"},
		},
	}

	for _, check := range checks {
		assertSensitive("resource", check.name, provider.ResourcesMap[check.name], check.resourceFields)
		assertSensitive("data source", check.name, provider.DataSourcesMap[check.name], check.dataFields)
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
