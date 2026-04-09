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
		ResourcesMap: map[string]*schema.Resource{
			"guacamole_user":                  guacamoleUser(),
			"guacamole_user_group":            guacamoleUserGroup(),
			"guacamole_connection_ssh":        guacamoleConnectionSSH(),
			"guacamole_connection_telnet":     guacamoleConnectionTelnet(),
			"guacamole_connection_rdp":        guacamoleConnectionRDP(),
			"guacamole_connection_vnc":        guacamoleConnectionVNC(),
			"guacamole_connection_kubernetes": guacamoleConnectionKubernetes(),
			"guacamole_connection_group":      guacamoleConnectionGroup(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"guacamole_user":                  dataSourceUser(),
			"guacamole_user_group":            dataSourceUserGroup(),
			"guacamole_connection_ssh":        dataSourceConnectionSSH(),
			"guacamole_connection_telnet":     dataSourceConnectionTelnet(),
			"guacamole_connection_rdp":        dataSourceConnectionRDP(),
			"guacamole_connection_vnc":        dataSourceConnectionVNC(),
			"guacamole_connection_kubernetes": dataSourceConnectionKubernetes(),
			"guacamole_connection_group":      dataSourceConnectionGroup(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	url := strings.TrimRight(d.Get("url").(string), "/")
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	token := d.Get("token").(string)
	data_source := d.Get("data_source").(string)
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
		DataSource:             data_source,
		Cookies:                cookies,
		DisableTLSVerification: disableTLS,
		DisableCookies:         disableCookies,
	}

	// Return a LazyClient that defers authentication until the first API call.
	// This allows Terraform to plan resources even when the Guacamole server
	// is not yet available (e.g., during HCP Terraform Stacks planning).
	return NewLazyClient(config), nil
}
