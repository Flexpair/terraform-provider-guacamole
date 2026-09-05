package guacamole

import (
	"context"

	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	guac "github.com/techBeck03/guacamole-api-client"
	types "github.com/techBeck03/guacamole-api-client/types"
)

func guacamoleUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Description: "Username of guacamole user",
				Required:    true,
				ForceNew:    true,
			},
			"password": {
				Type:        schema.TypeString,
				Description: "Password of guacamole user",
				Optional:    true,
				Sensitive:   true,
			},
			"last_active": {
				Type:        schema.TypeString,
				Description: "Epoch time string of last user activity",
				Computed:    true,
			},
			"attributes": {
				Type:        schema.TypeList,
				Description: "Attributes of guacamole user",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"organizational_role": {
							Type:        schema.TypeString,
							Description: "Organizational role of user",
							Optional:    true,
							Computed:    true,
						},
						"full_name": {
							Type:        schema.TypeString,
							Description: "Full name of user",
							Optional:    true,
							Computed:    true,
						},
						"email": {
							Type:        schema.TypeString,
							Description: "Email of user",
							Optional:    true,
							Computed:    true,
						},
						"expired": {
							Type:        schema.TypeBool,
							Description: "Whether the user is expired",
							Optional:    true,
							Computed:    false,
						},
						"timezone": {
							Type:        schema.TypeString,
							Description: "Timezone of user",
							Optional:    true,
							Computed:    true,
						},
						"access_window_start": {
							Type:        schema.TypeString,
							Description: "Access window start time for user",
							Optional:    true,
							Computed:    true,
						},
						"access_window_end": {
							Type:        schema.TypeString,
							Description: "Access window end time for user",
							Optional:    true,
							Computed:    true,
						},
						"disabled": {
							Type:        schema.TypeBool,
							Description: "Whether account is disabled",
							Optional:    true,
							Computed:    true,
						},
						"valid_from": {
							Type:        schema.TypeString,
							Description: "Start date for when user is valid",
							Optional:    true,
							Computed:    true,
						},
						"valid_until": {
							Type:        schema.TypeString,
							Description: "End date for when user is valid",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"group_membership": {
				Type:        schema.TypeSet,
				Description: "Groups this user is a member of",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"system_permissions": {
				Type:        schema.TypeSet,
				Description: "System permissions assigned to user",
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

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	check := validateUser(d)
	if check.HasError() {
		return check
	}

	user, err := convertResourceDataToGuacUser(d)

	if err != nil {
		return diag.FromErr(err)
	}

	err = client.CreateUser(&user)

	if err != nil {
		return diag.FromErr(err)
	}

	if check := applyUserCreatePermissions(d, user.Username, client); check.HasError() {
		diags = append(diags, check...)
		goto Cleanup
	}

	d.SetId(user.Username)
	return resourceUserRead(ctx, d, m)
Cleanup:
	d.SetId(user.Username)
	check = resourceUserDelete(ctx, d, m)
	if check.HasError() {
		diags = append(diags, check...)
	}
	return diags
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := readWithClient(m, d)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	userID := d.Id()
	user, err := client.ReadUser(userID)

	if err != nil {
		if isNotFoundError(err) {
			// The user no longer exists on the server (e.g. the gateway VM was
			// replaced). Clear the ID so Terraform plans a recreate instead of
			// failing the apply.
			d.SetId("")
			return diags
		}
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error reading guacamole user: " + userID,
			Detail:   err.Error(),
		})

		return diags
	}

	err = convertGuacUserToResourceData(d, &user)

	if err != nil {
		return diag.FromErr(err)
	}

	// Read group membership
	groups, err := client.GetUserGroupMembership(userID)

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("group_membership", groups)

	// Read system permissions
	permissions, err := client.GetUserPermissions(userID)

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
		connectionGroups = append(connectionGroups, group)
	}

	d.Set("connection_groups", connectionGroups)

	return diags
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	if d.HasChanges("username", "password", "last_active", "attributes") {
		check := validateUser(d)
		if check.HasError() {
			return check
		}

		user, err := convertResourceDataToGuacUser(d)
		if err != nil {
			return diag.FromErr(err)
		}
		err = client.UpdateUser(&user)

		if err != nil {
			return diag.FromErr(err)
		}
	}

	updates := userPermissionUpdates(client)
	if check := applyPermissionUpdates(d, d.Id(), updates); check.HasError() {
		return check
	}

	return resourceUserRead(ctx, d, m)
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	userID := d.Id()

	err := client.DeleteUser(userID)
	if err != nil && !isNotFoundError(err) {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func applyUserCreatePermissions(d *schema.ResourceData, username string, client *guac.Client) diag.Diagnostics {
	groupMembership := stringSetValues(d.Get("group_membership"))
	if check := applyPermissionItems(username, groupMembership, func(values []string) diag.Diagnostics { return validateGroups(client, values) }, client.NewAddGroupMemberPermission, client.SetUserGroupMembership); check.HasError() {
		return check
	}

	systemPermissions := stringSetValues(d.Get("system_permissions"))
	if check := applyPermissionItems(username, systemPermissions, func(values []string) diag.Diagnostics {
		return stringInSlice(types.SystemPermissions{}.ValidChoices(), values)
	}, client.NewAddSystemPermission, client.SetUserPermissions); check.HasError() {
		return check
	}

	return applyConnectionPermissionItems(username, stringSetValues(d.Get("connections")), stringSetValues(d.Get("connection_groups")), client.NewAddConnectionPermission, client.NewAddConnectionGroupPermission, client.SetUserPermissions)
}

func userPermissionUpdates(client *guac.Client) []permissionUpdateDefinition {
	return []permissionUpdateDefinition{
		{field: "group_membership", validate: func(values []string) diag.Diagnostics { return validateGroups(client, values) }, remove: client.NewRemoveGroupMemberPermission, add: client.NewAddGroupMemberPermission, apply: client.SetUserGroupMembership},
		{field: "system_permissions", validate: func(values []string) diag.Diagnostics {
			return stringInSlice(types.SystemPermissions{}.ValidChoices(), values)
		}, remove: client.NewRemoveSystemPermission, add: client.NewAddSystemPermission, apply: client.SetUserPermissions},
		{field: "connections", remove: client.NewRemoveConnectionPermission, add: client.NewAddConnectionPermission, apply: client.SetUserPermissions},
		{field: "connection_groups", remove: client.NewRemoveConnectionGroupPermission, add: client.NewAddConnectionGroupPermission, apply: client.SetUserPermissions},
	}
}

func convertResourceDataToGuacUser(d *schema.ResourceData) (types.GuacUser, error) {
	var user types.GuacUser

	user.Username = d.Get("username").(string)
	user.Password = d.Get("password").(string)

	attributeList := d.Get("attributes").([]interface{})

	if len(attributeList) > 0 {
		attributes := attributeList[0].(map[string]interface{})
		user.Attributes = types.GuacUserAttributes{
			GuacOrganizationalRole: attributes["organizational_role"].(string),
			GuacFullName:           attributes["full_name"].(string),
			Email:                  attributes["email"].(string),
			Expired:                boolToString(attributes["expired"].(bool)),
			Timezone:               attributes["timezone"].(string),
			AccessWindowStart:      attributes["access_window_start"].(string),
			AccessWindowEnd:        attributes["access_window_end"].(string),
			Disabled:               boolToString(attributes["disabled"].(bool)),
			ValidFrom:              attributes["valid_from"].(string),
			ValidUntil:             attributes["valid_until"].(string),
		}
	}

	return user, nil
}

func convertGuacUserToResourceData(d *schema.ResourceData, user *types.GuacUser) error {
	d.Set("username", user.Username)
	// Guacamole API never returns password on read (write-only field).
	// Preserve the configured value in state to prevent perpetual drift.
	if user.Password != "" {
		d.Set("password", user.Password)
	}
	d.Set("last_active", strconv.FormatInt(user.LastActive, 10))

	attributes := map[string]interface{}{
		"organizational_role": user.Attributes.GuacOrganizationalRole,
		"full_name":           user.Attributes.GuacFullName,
		"email":               user.Attributes.Email,
		"expired":             stringToBool(user.Attributes.Expired),
		"timezone":            user.Attributes.Timezone,
		"access_window_start": user.Attributes.AccessWindowStart,
		"access_window_end":   user.Attributes.AccessWindowEnd,
		"disabled":            stringToBool(user.Attributes.Disabled),
		"valid_from":          user.Attributes.ValidFrom,
		"valid_until":         user.Attributes.ValidUntil,
	}

	var attributeList []map[string]interface{}

	attributeList = append(attributeList, attributes)

	d.Set("attributes", attributeList)

	return nil
}

func validateGroups(client *guac.Client, groups []string) diag.Diagnostics {
	var diags diag.Diagnostics
	var invalidUserGroups []string

	userGroups, err := client.ListUserGroups()
	if err != nil {
		return diag.FromErr(err)
	}
	for _, group := range groups {
		matchFlag := false
		for _, g := range userGroups {
			if group == g.Identifier {
				matchFlag = true
				break
			}
		}
		if !matchFlag {
			invalidUserGroups = append(invalidUserGroups, group)
		}
	}
	if len(invalidUserGroups) > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Invalid user group(s) supplied",
			Detail:   "The following groups are invalid for group_membership: " + strings.Join(invalidUserGroups[:], ", "),
		})
		return diags
	}
	return diags
}

func validateUser(d *schema.ResourceData) diag.Diagnostics {
	var diags diag.Diagnostics
	// validate attributes
	attributeList := d.Get("attributes").([]interface{})

	if len(attributeList) > 0 {
		attributes := attributeList[0].(map[string]interface{})

		// validate timezone string
		timezone := attributes["timezone"].(string)
		_, err := time.LoadLocation(timezone)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid timezone",
				Detail:   "Unable to process timezone string: " + timezone,
			})
		}

		validFrom, changed := d.GetOk("attributes.0.valid_from")
		if changed {
			check := validateTimestring(validFrom.(string), "valid_from")
			if check.HasError() {
				diags = append(diags, check...)
			}
		}

		validUntil, changed := d.GetOk("attributes.0.valid_until")
		if changed {
			check := validateTimestring(validUntil.(string), "valid_until")
			if check.HasError() {
				diags = append(diags, check...)
			}
		}
	}
	return diags
}
