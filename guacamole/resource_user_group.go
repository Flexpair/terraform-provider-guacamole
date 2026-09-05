package guacamole

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	guac "github.com/techBeck03/guacamole-api-client"
	types "github.com/techBeck03/guacamole-api-client/types"
)

func guacamoleUserGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserGroupCreate,
		ReadContext:   resourceUserGroupRead,
		UpdateContext: resourceUserGroupUpdate,
		DeleteContext: resourceUserGroupDelete,
		Schema: map[string]*schema.Schema{
			"identifier": {
				Type:        schema.TypeString,
				Description: "Identifier of guacamole user group",
				Required:    true,
				ForceNew:    true,
			},
			"attributes": {
				Type:        schema.TypeList,
				Description: "Attributes of guacamole user group",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disabled": {
							Type:        schema.TypeBool,
							Description: "Whether group is disabled",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"group_membership": {
				Type:        schema.TypeSet,
				Description: "Groups this user group is a member of",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"system_permissions": {
				Type:        schema.TypeSet,
				Description: "System permissions assigned to user group",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"connections": {
				Type:        schema.TypeSet,
				Description: "Connections identifiers a user has permission to read",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"connection_groups": {
				Type:        schema.TypeSet,
				Description: "Connection Group identifiers a user has permission to read",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceUserGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	group, err := convertResourceDataToGuacUserGroup(d)

	if err != nil {
		return diag.FromErr(err)
	}

	err = client.CreateUserGroup(&group)

	if err != nil {
		return diag.FromErr(err)
	}

	if check := applyUserGroupCreatePermissions(d, group.Identifier, client); check.HasError() {
		diags = append(diags, check...)
		goto Cleanup
	}

	d.SetId(group.Identifier)
	resourceUserGroupRead(ctx, d, m)

	return diags
Cleanup:
	d.SetId(group.Identifier)
	check := resourceUserGroupDelete(ctx, d, m)
	if check.HasError() {
		diags = append(diags, check...)
	}
	return diags
}

func resourceUserGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := readWithClient(m, d)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	identifier := d.Id()
	group, err := client.ReadUserGroup(identifier)

	if err != nil {
		if isNotFoundError(err) {
			// The user group no longer exists on the server (e.g. the gateway
			// VM was replaced). Clear the ID so Terraform plans a recreate
			// instead of failing the apply.
			d.SetId("")
			return diags
		}
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error reading guacamole user group: " + identifier,
			Detail:   err.Error(),
		})

		return diags
	}

	err = convertGuacUserGroupToResourceData(d, &group)

	if err != nil {
		return diag.FromErr(err)
	}

	// Read group membership
	groups, err := client.GetUserGroupMemberGroups(identifier)

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("group_membership", groups)

	// Read system permissions
	permissions, err := client.GetUserGroupPermissions(identifier)

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("system_permissions", permissions.SystemPermissions)

	// Get connections
	var connections []string
	for connection := range permissions.ConnectionPermissions {
		connections = append(connections, connection)
	}

	d.Set("connections", connections)

	// Get connection groups
	var connectionGroups []string
	for group := range permissions.ConnectionGroupPermissions {
		connections = append(connectionGroups, group)
	}

	d.Set("connection_groups", connectionGroups)

	return diags
}

func resourceUserGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	if d.HasChanges("username", "attributes") {
		group, err := convertResourceDataToGuacUserGroup(d)
		if err != nil {
			return diag.FromErr(err)
		}
		err = client.UpdateUserGroup(&group)

		if err != nil {
			return diag.FromErr(err)
		}
	}

	updates := userGroupPermissionUpdates(client)
	if check := applyPermissionUpdates(d, d.Id(), updates); check.HasError() {
		return check
	}

	return resourceUserGroupRead(ctx, d, m)
}

func resourceUserGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	identifier := d.Id()

	err := client.DeleteUserGroup(identifier)
	if err != nil && !isNotFoundError(err) {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func applyUserGroupCreatePermissions(d *schema.ResourceData, identifier string, client *guac.Client) diag.Diagnostics {
	groupMembership := stringSetValues(d.Get("group_membership"))
	if check := applyPermissionItems(identifier, groupMembership, func(values []string) diag.Diagnostics { return validateGroups(client, values) }, client.NewAddGroupMemberPermission, client.SetUserGroupMemberGroups); check.HasError() {
		return check
	}

	systemPermissions := stringSetValues(d.Get("system_permissions"))
	if check := applyPermissionItems(identifier, systemPermissions, func(values []string) diag.Diagnostics {
		return stringInSlice(types.SystemPermissions{}.ValidChoices(), values)
	}, client.NewAddSystemPermission, client.SetUserGroupPermissions); check.HasError() {
		return check
	}

	return applyConnectionPermissionItems(identifier, stringSetValues(d.Get("connections")), stringSetValues(d.Get("connection_groups")), client.NewAddConnectionPermission, client.NewAddConnectionGroupPermission, client.SetUserGroupPermissions)
}

func userGroupPermissionUpdates(client *guac.Client) []permissionUpdateDefinition {
	return []permissionUpdateDefinition{
		{field: "group_membership", validate: func(values []string) diag.Diagnostics { return validateGroups(client, values) }, remove: client.NewRemoveGroupMemberPermission, add: client.NewAddGroupMemberPermission, apply: client.SetUserGroupMemberGroups},
		{field: "system_permissions", validate: func(values []string) diag.Diagnostics {
			return stringInSlice(types.SystemPermissions{}.ValidChoices(), values)
		}, remove: client.NewRemoveSystemPermission, add: client.NewAddSystemPermission, apply: client.SetUserGroupPermissions},
		{field: "connections", remove: client.NewRemoveConnectionPermission, add: client.NewAddConnectionPermission, apply: client.SetUserGroupPermissions},
		{field: "connection_groups", remove: client.NewRemoveConnectionGroupPermission, add: client.NewAddConnectionGroupPermission, apply: client.SetUserGroupPermissions},
	}
}

func convertResourceDataToGuacUserGroup(d *schema.ResourceData) (types.GuacUserGroup, error) {
	var group types.GuacUserGroup

	group.Identifier = d.Get("identifier").(string)

	attributeList := d.Get("attributes").([]interface{})

	if len(attributeList) > 0 {
		attributes := attributeList[0].(map[string]interface{})
		group.Attributes = types.GuacUserGroupAttributes{
			Disabled: boolToString(attributes["disabled"].(bool)),
		}
	}

	return group, nil
}

func convertGuacUserGroupToResourceData(d *schema.ResourceData, group *types.GuacUserGroup) error {
	d.Set("identifier", group.Identifier)

	attributes := map[string]interface{}{
		"disabled": stringToBool(group.Attributes.Disabled),
	}

	var attributeList []map[string]interface{}

	attributeList = append(attributeList, attributes)

	d.Set("attributes", attributeList)

	return nil
}
