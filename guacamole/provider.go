package guacamole

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	guac "github.com/techBeck03/guacamole-api-client"
)

// Provider -
func Provider() *schema.Provider {
	resources := map[string]*schema.Resource{
		"guacamole_user":       guacamoleUser(),
		"guacamole_user_group": guacamoleUserGroup(),
	}
	for name, resource := range connectionSchemas(false) {
		resources[name] = resource
	}

	dataSources := map[string]*schema.Resource{
		"guacamole_user":       dataSourceUser(),
		"guacamole_user_group": dataSourceUserGroup(),
	}
	for name, dataSource := range connectionSchemas(true) {
		dataSources[name] = dataSource
	}

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GUACAMOLE_URL", nil),
			},
			"username": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"password"},
				DefaultFunc:  schema.EnvDefaultFunc("GUACAMOLE_USERNAME", nil),
			},
			"password": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"username"},
				AtLeastOneOf: []string{"password", "token"},
				Sensitive:    true,
				DefaultFunc:  schema.EnvDefaultFunc("GUACAMOLE_PASSWORD", nil),
			},
			"token": {
				Type:         schema.TypeString,
				Optional:     true,
				AtLeastOneOf: []string{"password", "token"},
				Sensitive:    true,
				DefaultFunc:  schema.EnvDefaultFunc("GUACAMOLE_TOKEN", nil),
			},
			"data_source": {
				Type:             schema.TypeString,
				Optional:         true,
				RequiredWith:     []string{"token"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"postgresql", "mysql"}, true)),
				DefaultFunc:      schema.EnvDefaultFunc("GUACAMOLE_DATA_SOURCE", nil),
			},
			"cookies": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"disable_tls_verification": {
				Type:        schema.TypeBool,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GUACAMOLE_DISABLE_TLS", false),
			},
			"disable_cookies": {
				Type:        schema.TypeBool,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GUACAMOLE_DISABLE_COOKIES", false),
			},
		},
		ResourcesMap:         resources,
		DataSourcesMap:       dataSources,
		ConfigureContextFunc: providerConfigure,
	}
}

var (
	sshResourceFields          = []string{"password", "private_key", "passphrase"}
	sshDataSourceFields        = []string{"private_key", "passphrase"}
	rdpCredentialFields        = []string{"password", "gateway_password", "sftp_password", "sftp_private_key", "sftp_passphrase"}
	vncCredentialFields        = []string{"password", "sftp_password", "sftp_private_key", "sftp_passphrase"}
	kubernetesCredentialFields = []string{"client_cert", "client_key"}
)

type connectionSchemaDefinition struct {
	name             string
	resource         func() *schema.Resource
	dataSource       func() *schema.Resource
	resourceFields   []string
	dataSourceFields []string
}

var connectionSchemaDefinitions = []connectionSchemaDefinition{
	{
		name:             "guacamole_connection_ssh",
		resource:         guacamoleConnectionSSH,
		dataSource:       dataSourceConnectionSSH,
		resourceFields:   sshResourceFields,
		dataSourceFields: sshDataSourceFields,
	},
	{
		name:             "guacamole_connection_telnet",
		resource:         guacamoleConnectionTelnet,
		dataSource:       dataSourceConnectionTelnet,
		resourceFields:   []string{"password"},
		dataSourceFields: nil,
	},
	{
		name:             "guacamole_connection_rdp",
		resource:         guacamoleConnectionRDP,
		dataSource:       dataSourceConnectionRDP,
		resourceFields:   rdpCredentialFields,
		dataSourceFields: rdpCredentialFields,
	},
	{
		name:             "guacamole_connection_vnc",
		resource:         guacamoleConnectionVNC,
		dataSource:       dataSourceConnectionVNC,
		resourceFields:   vncCredentialFields,
		dataSourceFields: vncCredentialFields,
	},
	{
		name:             "guacamole_connection_kubernetes",
		resource:         guacamoleConnectionKubernetes,
		dataSource:       dataSourceConnectionKubernetes,
		resourceFields:   kubernetesCredentialFields,
		dataSourceFields: kubernetesCredentialFields,
	},
	{
		name:       "guacamole_connection_group",
		resource:   guacamoleConnectionGroup,
		dataSource: dataSourceConnectionGroup,
	},
}

func connectionSchemas(dataSource bool) map[string]*schema.Resource {
	result := make(map[string]*schema.Resource, len(connectionSchemaDefinitions))
	for _, definition := range connectionSchemaDefinitions {
		resource := definition.resource()
		fields := definition.resourceFields
		if dataSource {
			resource = definition.dataSource()
			fields = definition.dataSourceFields
		}
		result[definition.name] = markConnectionParametersSensitive(resource, fields...)
	}
	return result
}

func markConnectionParametersSensitive(resource *schema.Resource, fields ...string) *schema.Resource {
	if len(fields) == 0 {
		return resource
	}
	parameters := resource.Schema["parameters"].Elem.(*schema.Resource).Schema
	for _, field := range fields {
		parameters[field].Sensitive = true
	}
	return resource
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	url := strings.TrimRight(d.Get("url").(string), "/")
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	token := d.Get("token").(string)
	dataSource := d.Get("data_source").(string)
	disableTLS := d.Get("disable_tls_verification").(bool)
	disableCookies := d.Get("disable_cookies").(bool)

	cookies := make(map[string]string)
	cookieMap := d.Get("cookies").(map[string]interface{})
	if len(cookieMap) > 0 {
		for k, v := range cookieMap {
			cookies[k] = v.(string)
		}
	}

	config := guac.Config{
		URL:                    url,
		Username:               username,
		Password:               password,
		Token:                  token,
		DataSource:             dataSource,
		Cookies:                cookies,
		DisableTLSVerification: disableTLS,
		DisableCookies:         disableCookies,
	}

	// Return a LazyClient that defers authentication until the first API call.
	// This allows Terraform to plan resources even when the Guacamole server
	// is not yet available (e.g., during HCP Terraform Stacks planning).
	return NewLazyClient(config), nil
}
