package guacamole

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	guac "github.com/techBeck03/guacamole-api-client"
	types "github.com/techBeck03/guacamole-api-client/types"
)

func guacamoleConnectionRDP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceConnectionRDPCreate,
		ReadContext:   resourceConnectionRDPRead,
		UpdateContext: resourceConnectionRDPUpdate,
		DeleteContext: resourceConnectionRDPDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the guacamole connection",
				Required:    true,
			},
			"identifier": {
				Type:        schema.TypeString,
				Description: "Numeric identifier of the guacamole connection",
				Computed:    true,
			},
			"parent_identifier": {
				Type:        schema.TypeString,
				Description: "Parent identifier of the guacamole connection",
				Optional:    true,
				Default:     "ROOT",
			},
			"protocol": {
				Type:        schema.TypeString,
				Description: "Protocol of the guacamole connection",
				Optional:    true,
				Default:     "rdp",
				ForceNew:    true,
			},
			"attributes": {
				Type:        schema.TypeList,
				Description: "Attributes of the guacamole connection",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"max_connections": {Type: schema.TypeString, Optional: true, Computed: true},
				}},
			},
			"parameters": {
				Type:        schema.TypeList,
				Description: "Parameters of the guacamole connection",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"port":             {Type: schema.TypeString, Optional: true, Computed: true},
					"gateway_port":     {Type: schema.TypeString, Optional: true, Computed: true},
					"width":            {Type: schema.TypeString, Optional: true, Computed: true},
					"height":           {Type: schema.TypeString, Optional: true, Computed: true},
					"dpi":              {Type: schema.TypeString, Optional: true, Computed: true},
					"preconnection_id": {Type: schema.TypeString, Optional: true, Computed: true},
					"sftp_port":        {Type: schema.TypeString, Optional: true, Computed: true},
				}},
			},
		},
	}
}

func resourceConnectionRDPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}
	if check := validateConnectionRDP(d, client); check.HasError() {
		return check
	}
	connection, check := convertResourceDataToGuacConnectionRDP(d)
	if check.HasError() {
		return check
	}
	if err := client.CreateConnection(&connection); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(connection.Identifier)
	return resourceConnectionRDPRead(ctx, d, m)
}

func resourceConnectionRDPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := readWithClient(m, d)
	if client == nil {
		return diags
	}
	connection, err := client.ReadConnection(d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return diags
		}
		return append(diags, diag.Diagnostic{Severity: diag.Error, Summary: "Error reading guacamole connection: " + d.Id(), Detail: err.Error()})
	}
	check := convertGuacConnectionRDPToResourceData(d, &connection)
	return append(diags, check...)
}

func resourceConnectionRDPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}
	if check := validateConnectionRDP(d, client); check.HasError() {
		return check
	}
	connection, check := convertResourceDataToGuacConnectionRDP(d)
	if check.HasError() {
		return check
	}
	if err := client.UpdateConnection(&connection); err != nil {
		return diag.FromErr(err)
	}
	return resourceConnectionRDPRead(ctx, d, m)
}

func resourceConnectionRDPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}
	if err := client.DeleteConnection(d.Id()); err != nil && !isNotFoundError(err) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}

func validateConnectionRDP(d *schema.ResourceData, _ *guac.Client) diag.Diagnostics {
	var diags diag.Diagnostics
	var parameterInterface types.GuacConnectionParameters
	var attributeInterface types.GuacConnectionAttributes
	attributeList := d.Get("attributes").([]interface{})
	restrictedValueAttributes := map[string][]string{
		"guacd_encryption": attributeInterface.ValidEncryptionTypes(),
	}
	if len(attributeList) > 0 {
		diags = append(diags, validateStringFields(attributeList, []string{
			"guacd_port", "weight", "max_connections", "max_connections_per_user",
		}, restrictedValueAttributes, "attribute")...)
	}
	parameterList := d.Get("parameters").([]interface{})
	if len(parameterList) == 0 {
		return diags
	}
	parameters := parameterList[0].(map[string]interface{})
	parameterDiagnostics := validateStringFields([]interface{}{parameters}, []string{
		"port", "gateway_port", "width", "height", "dpi", "preconnection_id", "sftp_port", "sftp_keepalive_interval", "wol_boot_wait_time",
	}, map[string][]string{
		"security_mode": parameterInterface.ValidSecurityModes(),
		"keyboard_layout": parameterInterface.ValidKeyboardLayouts(),
		"color_depth": parameterInterface.ValidColorDepths(),
		"resize_method": parameterInterface.ValidResizeMethods(),
	}, "parameter")
	diags = append(diags, validateTimezone(parameterList, "timezone")...)
	return append(diags, parameterDiagnostics...)
}

func convertResourceDataToGuacConnectionRDP(d *schema.ResourceData) (types.GuacConnection, diag.Diagnostics) {
	return types.GuacConnection{}, nil
}

func convertGuacConnectionRDPToResourceData(d *schema.ResourceData, connection *types.GuacConnection) diag.Diagnostics {
	return nil
}
