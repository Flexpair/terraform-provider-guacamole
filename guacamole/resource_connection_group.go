package guacamole

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	types "github.com/techBeck03/guacamole-api-client/types"
)

func guacamoleConnectionGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceConnectionGroupCreate,
		ReadContext:   resourceConnectionGroupRead,
		UpdateContext: resourceConnectionGroupUpdate,
		DeleteContext: resourceConnectionGroupDelete,
		Schema: map[string]*schema.Schema{
			"identifier": {
				Type:        schema.TypeString,
				Description: "Identifier of guacamole connection group",
				Computed:    true,
			},
			"parent_identifier": {
				Type:        schema.TypeString,
				Description: "Parent Identifier of guacamole connection group",
				Required:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of guacamole connection group",
				Required:    true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of guacamole connection group",
				Optional:    true,
				Default:     "ORGANIZATIONAL",
				StateFunc: func(val interface{}) string {
					return strings.ToUpper(val.(string))
				},
			},
			"active_connections": {
				Type:        schema.TypeInt,
				Description: "Active connections of guacamole connection group",
				Computed:    true,
			},
			"attributes": {
				Type:        schema.TypeList,
				Description: "Attributes of guacamole connection group",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_connections": {
							Type:        schema.TypeString,
							Description: "Maximum number of total simultaneous connections allowed",
							Optional:    true,
							Computed:    true,
						},
						"max_connections_per_user": {
							Type:        schema.TypeString,
							Description: "Maximum number of simultaneous connections allowed per user",
							Optional:    true,
							Computed:    true,
						},
						"enable_session_affinity": {
							Type:        schema.TypeBool,
							Description: "Enable session affinity",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceConnectionGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	validate := validateConnectionGroup(d)

	if validate.HasError() {
		return validate
	}

	group, check := convertResourceDataToGuacConnectionGroup(d)

	if check.HasError() {
		return check
	}

	err := client.CreateConnectionGroup(&group)

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("identifier", group.Identifier)
	d.SetId(group.Identifier)

	if diags.HasError() {
		return diags
	}

	return resourceConnectionGroupRead(ctx, d, m)
}

func resourceConnectionGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := readWithClient(m, d)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	identifier := d.Id()

	group, err := client.ReadConnectionGroup(identifier)

	if err != nil {
		if isNotFoundError(err) {
			// The connection group no longer exists on the server (e.g. the
			// gateway VM was replaced). Clear the ID so Terraform plans a
			// recreate instead of failing the apply.
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	check := convertGuacConnectionGroupToResourceData(d, &group)
	if check.HasError() {
		return check
	}

	d.SetId(identifier)

	return diags
}

func resourceConnectionGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	if d.HasChanges("name", "identifier", "parent_identifier", "type", "attributes") {
		validate := validateConnectionGroup(d)

		if validate.HasError() {
			return validate
		}

		group, check := convertResourceDataToGuacConnectionGroup(d)

		if check.HasError() {
			return check
		}

		err := client.UpdateConnectionGroup(&group)

		if err != nil {
			return diag.FromErr(err)
		}

		d.SetId(group.Identifier)

	} else {
		d.SetId(d.Id())
	}

	if diags.HasError() {
		return diags
	}

	return resourceConnectionGroupRead(ctx, d, m)
}

func resourceConnectionGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	identifier := d.Id()

	err := client.DeleteConnectionGroup(identifier)
	if err != nil && !isNotFoundError(err) {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func convertResourceDataToGuacConnectionGroup(d *schema.ResourceData) (types.GuacConnectionGroup, diag.Diagnostics) {
	var group types.GuacConnectionGroup
	var diags diag.Diagnostics

	group.Identifier = d.Get("identifier").(string)
	group.ParentIdentifier = d.Get("parent_identifier").(string)
	group.Name = d.Get("name").(string)
	group.Type = strings.ToUpper(d.Get("type").(string))

	attributeList := d.Get("attributes").([]interface{})

	if len(attributeList) > 0 {
		attributes := attributeList[0].(map[string]interface{})
		group.Attributes = types.GuacConnectionGroupAttributes{
			MaxConnections:        attributes["max_connections"].(string),
			MaxConnectionsPerUser: attributes["max_connections_per_user"].(string),
			EnableSessionAffinity: boolToString(attributes["enable_session_affinity"].(bool)),
		}
	}

	return group, diags
}

func convertGuacConnectionGroupToResourceData(d *schema.ResourceData, group *types.GuacConnectionGroup) diag.Diagnostics {
	d.Set("identifier", group.Identifier)
	d.Set("parent_identifier", group.ParentIdentifier)
	d.Set("name", group.Name)
	d.Set("type", group.Type)

	attributes := map[string]interface{}{
		"max_connections":          group.Attributes.MaxConnections,
		"max_connections_per_user": group.Attributes.MaxConnectionsPerUser,
		"enable_session_affinity":  stringToBool(group.Attributes.EnableSessionAffinity),
	}
	var attributeList []map[string]interface{}

	attributeList = append(attributeList, attributes)

	d.Set("attributes", attributeList)

	return nil
}

