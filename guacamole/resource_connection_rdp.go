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
				Description: "Protocol type of the guacamole connection",
				Computed:    true,
			},
			"active_connections": {
				Type:        schema.TypeInt,
				Description: "Active connection count for the guacamole connection",
				Computed:    true,
			},
			"attributes": {
				Type:        schema.TypeList,
				Description: "Guacamole connection attributes",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"guacd_hostname": {
							Type:        schema.TypeString,
							Description: "Guacd proxy hostname",
							Optional:    true,
							Computed:    true,
						},
						"guacd_port": {
							Type:        schema.TypeString,
							Description: "Guacd proxy port",
							Optional:    true,
							Computed:    true,
						},
						"guacd_encryption": {
							Type:        schema.TypeString,
							Description: "Guacd proxy encryption type",
							Optional:    true,
							Computed:    true,
						},
						"failover_only": {
							Type:        schema.TypeBool,
							Description: "Use load balancing for failover only",
							Optional:    true,
							Computed:    true,
						},
						"weight": {
							Type:        schema.TypeString,
							Description: "Load balancing connection weight",
							Optional:    true,
							Computed:    true,
						},
						"max_connections": {
							Type:        schema.TypeString,
							Description: "Maximum concurrent total connections",
							Optional:    true,
							Computed:    true,
						},
						"max_connections_per_user": {
							Type:        schema.TypeString,
							Description: "Maximum concurrent connections per user",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"parameters": {
				Type:        schema.TypeList,
				Description: "Guacamole connection parameters",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hostname": {
							Type:        schema.TypeString,
							Description: "Hostname of target",
							Required:    true,
						},
						"port": {
							Type:        schema.TypeString,
						Description: "Port for target connection",
						Optional:    true,
						Computed:    true,
						},
						"username": {
							Type:        schema.TypeString,
						Description: "Username for rdp connection",
						Required:    true,
						},
						"password": {
							Type:        schema.TypeString,
						Description: "Password for rdp connection",
						Optional:    true,
						Computed:    true,
						},
						"domain": {
							Type:        schema.TypeString,
						Description: "Domain name of rdp connection",
							Optional:    true,
							Computed:    true,
						},
						"security_mode": {
							Type:        schema.TypeString,
							Description: "RDP security mode",
							Optional:    true,
							Computed:    true,
						},
						"disable_authentication": {
							Type:        schema.TypeBool,
							Description: "Disable rdp authentication",
							Optional:    true,
							Computed:    true,
						},
						"ignore_cert": {
							Type:        schema.TypeBool,
							Description: "Ignore server certificate",
							Optional:    true,
							Computed:    true,
						},
						"gateway_hostname": {
							Type:        schema.TypeString,
							Description: "Gateway hostname",
							Optional:    true,
							Computed:    true,
						},
						"gateway_port": {
							Type:        schema.TypeString,
							Description: "Gateway port",
							Optional:    true,
							Computed:    true,
						},
						"gateway_username": {
							Type:        schema.TypeString,
							Description: "Gateway username",
							Optional:    true,
							Computed:    true,
						},
						"gateway_password": {
							Type:        schema.TypeString,
							Description: "Gateway password",
							Optional:    true,
							Computed:    true,
						},
						"gateway_domain": {
							Type:        schema.TypeString,
							Description: "Gateway domain",
							Optional:    true,
							Computed:    true,
						},
						"initial_program": {
							Type:        schema.TypeString,
						Description: "Initial program",
							Optional:    true,
							Computed:    true,
						},
						"client_name": {
							Type:        schema.TypeString,
						Description: "Client name",
							Optional:    true,
							Computed:    true,
						},
						"keyboard_layout": {
							Type:        schema.TypeString,
						Description: "Keyboard layout",
							Optional:    true,
							Computed:    true,
						},
						"timezone": {
							Type:        schema.TypeString,
						Description: "Timezone",
							Optional:    true,
							Computed:    true,
						},
						"administrator_console": {
							Type:        schema.TypeBool,
						Description: "Administrator console",
							Optional:    true,
							Computed:    true,
						},
						"width": {
							Type:        schema.TypeString,
							Description: "RDP width",
							Optional:    true,
							Computed:    true,
						},
						"height": {
							Type:        schema.TypeString,
						Description: "RDP height",
							Optional:    true,
							Computed:    true,
						},
						"dpi": {
							Type:        schema.TypeString,
							Description: "RDP DPI",
							Optional:    true,
							Computed:    true,
						},
						"color_depth": {
							Type:        schema.TypeString,
							Description: "RDP color depth",
							Optional:    true,
							Computed:    true,
						},
						"disable_audio": {
							Type:        schema.TypeBool,
							Description: "Disable audio",
							Optional:    true,
							Computed:    true,
						},
						"enable_audio_input": {
							Type:        schema.TypeBool,
						Description: "Enable audio input",
							Optional:    true,
							Computed:    true,
						},
						"enable_printing": {
							Type:        schema.TypeBool,
						Description: "Enable printing",
							Optional:    true,
							Computed:    true,
						},
						"enable_drive": {
							Type:        schema.TypeBool,
							Description: "Enable drive",
							Optional:    true,
							Computed:    true,
						},
						"drive_name": {
							Type:        schema.TypeString,
							Description: "Drive name",
							Optional:    true,
							Computed:    true,
						},
						"drive_path": {
							Type:        schema.TypeString,
							Description: "Drive path",
							Optional:    true,
							Computed:    true,
						},
						"create_drive_path": {
							Type:        schema.TypeBool,
							Description: "Create drive path",
							Optional:    true,
							Computed:    true,
						},
						"console": {
							Type:        schema.TypeBool,
							Description: "Console mode",
							Optional:    true,
							Computed:    true,
						},
						"console_audio": {
							Type:        schema.TypeBool,
							Description: "Console audio",
							Optional:    true,
							Computed:    true,
						},
						"preconnection_id": {
							Type:        schema.TypeString,
							Description: "Preconnection ID",
							Optional:    true,
							Computed:    true,
						},
						"remote_app": {
							Type:        schema.TypeString,
							Description: "Remote app",
							Optional:    true,
							Computed:    true,
						},
						"remote_app_name": {
							Type:        schema.TypeString,
						Description: "Remote app name",
							Optional:    true,
							Computed:    true,
						},
						"remote_app_dir": {
							Type:        schema.TypeString,
						Description: "Remote app directory",
							Optional:    true,
							Computed:    true,
						},
						"remote_app_args": {
							Type:        schema.TypeString,
							Description: "Remote app arguments",
							Optional:    true,
							Computed:    true,
						},
						"load_balance_info": {
							Type:        schema.TypeString,
							Description: "Load balance info",
							Optional:    true,
							Computed:    true,
						},
						"disable_copy": {
							Type:        schema.TypeBool,
						Description: "Disable copy",
							Optional:    true,
							Computed:    true,
						},
						"disable_paste": {
							Type:        schema.TypeBool,
						Description: "Disable paste",
							Optional:    true,
							Computed:    true,
						},
						"read_only": {
							Type:        schema.TypeBool,
						Description: "Read-only mode",
							Optional:    true,
							Computed:    true,
						},
						"resize_method": {
							Type:        schema.TypeString,
						Description: "Resize method",
						Optional:    true,
						Computed:    true,
						},
						"enable_sftp": {
							Type:        schema.TypeBool,
						Description: "Enable SFTP",
						Optional:    true,
						Computed:    true,
						},
						"sftp_hostname": {
							Type:        schema.TypeString,
						Description: "SFTP hostname",
							Optional:    true,
						Computed:    true,
						},
						"sftp_port": {
							Type:        schema.TypeString,
						Description: "SFTP port",
						Optional:    true,
						Computed:    true,
						},
						"sftp_username": {
							Type:        schema.TypeString,
						Description: "SFTP username",
						Optional:    true,
						Computed:    true,
						},
						"sftp_password": {
							Type:        schema.TypeString,
						Description: "SFTP password",
						Optional:    true,
						Computed:    true,
						},
						"sftp_private_key": {
							Type:        schema.TypeString,
						Description: "SFTP private key",
						Optional:    true,
						Computed:    true,
						},
						"sftp_passphrase": {
							Type:        schema.TypeString,
						Description: "SFTP passphrase",
						Optional:    true,
						Computed:    true,
						},
						"sftp_host_key": {
							Type:        schema.TypeString,
							Description: "SFTP host key",
						Optional:    true,
						Computed:    true,
						},
						"sftp_directory": {
							Type:        schema.TypeString,
						Description: "SFTP directory",
						Optional:    true,
						Computed:    true,
						},
						"sftp_root_directory": {
							Type:        schema.TypeString,
						Description: "SFTP root directory",
							Optional:    true,
							Computed:    true,
						},
						"sftp_disable_download": {
							Type:        schema.TypeBool,
						Description: "Disable SFTP download",
							Optional:    true,
							Computed:    true,
						},
						"sftp_disable_upload": {
							Type:        schema.TypeBool,
						Description: "Disable SFTP upload",
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

func resourceConnectionRDPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := readWithClient(m, d)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	identifier := d.Id()

	connection, err := client.ReadConnection(identifier)

	if err != nil {
		if isNotFoundError(err) {
			// The connection no longer exists on the server (e.g. the gateway
			// VM was replaced). Clear the ID so Terraform plans a recreate
			// instead of failing the apply.
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	check := convertGuacConnectionRDPToResourceData(d, &connection)
	if check.HasError() {
		return check
	}

	d.SetId(identifier)

	return diags
}

func resourceConnectionRDPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	validate := validateConnectionRDP(d, client)

	if validate.HasError() {
		return validate
	}

	connection, check := convertResourceDataToGuacConnectionRDP(d)

	if check.HasError() {
		return check
	}

	err := client.CreateConnection(&connection)

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("identifier", connection.Identifier)
	d.SetId(connection.Identifier)

	if diags.HasError() {
		return diags
	}

	return resourceConnectionRDPRead(ctx, d, m)
}

func resourceConnectionRDPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	if d.HasChanges("name", "identifier", "parent_identifier", "attributes", "parameters") {
		validate := validateConnectionRDP(d, client)

		if validate.HasError() {
			return validate
		}

		connection, check := convertResourceDataToGuacConnectionRDP(d)

		if check.HasError() {
			return check
		}

		err := client.UpdateConnection(&connection)

		if err != nil {
			return diag.FromErr(err)
		}

		d.SetId(connection.Identifier)

	} else {
		d.SetId(d.Id())
	}

	if diags.HasError() {
		return diags
	}

	return resourceConnectionRDPRead(ctx, d, m)
}

func resourceConnectionRDPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, diags := writeWithClient(m)
	if client == nil {
		return diags
	}

	// Warning or errors can be collected in a slice type

	err := client.DeleteConnection(d.Id())

	if err != nil && !isNotFoundError(err) {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func convertGuacConnectionRDPToResourceData(d *schema.ResourceData, connection *types.GuacConnection) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	d.Set("name", connection.Name)
	d.Set("identifier", connection.Identifier)
	d.Set("parent_identifier", connection.ParentIdentifier)
	d.Set("protocol", connection.Protocol)
	d.Set("active_connections", connection.ActiveConnections)

	attributes := map[string]interface{}{
		"guacd_hostname":           connection.Attributes.GuacdHostname,
		"guacd_port":               connection.Attributes.GuacdPort,
		"guacd_encryption":         connection.Attributes.GuacdEncryption,
		"failover_only":            stringToBool(connection.Attributes.FailoverOnly),
		"weight":                   connection.Attributes.Weight,
		"max_connections":          connection.Attributes.MaxConnections,
		"max_connections_per_user": connection.Attributes.MaxConnectionsPerUser,
	}
	var attributeList []map[string]interface{}

	attributeList = append(attributeList, attributes)

	d.Set("attributes", attributeList)

	parameters := map[string]interface{}{
		"hostname":                     connection.Parameters.Hostname,
		"port":                         connection.Parameters.Port,
		"username":                     connection.Parameters.Username,
		"password":                     connection.Parameters.Password,
		"domain":                       connection.Parameters.Domain,
		"security_mode":                connection.Parameters.Security,
		"disable_authentication":       stringToBool(connection.Parameters.DisableAuthentication),
		"ignore_cert":                  stringToBool(connection.Parameters.IgnoreCert),
		"gateway_hostname":             connection.Parameters.GatewayHostname,
		"gateway_port":                 connection.Parameters.GatewayPort,
		"gateway_username":             connection.Parameters.GatewayUsername,
		"gateway_password":             connection.Parameters.GatewayPassword,
		"gateway_domain":               connection.Parameters.GatewayDomain,
		"initial_program":              connection.Parameters.InitialProgram,
		"client_name":                  connection.Parameters.ClientName,
		"keyboard_layout":              connection.Parameters.KeyboardLayout,
		"timezone":                     connection.Parameters.Timezone,
		"administrator_console":        stringToBool(connection.Parameters.AdministratorConsole),
		"width":                        connection.Parameters.Width,
		"height":                       connection.Parameters.Height,
		"dpi":                          connection.Parameters.DPI,
		"color_depth":                  connection.Parameters.ColorDepth,
		"disable_audio":                stringToBool(connection.Parameters.DisableAudio),
		"enable_audio_input":           stringToBool(connection.Parameters.EnableAudioInput),
		"enable_printing":              stringToBool(connection.Parameters.EnablePrinting),
		"enable_drive":                 stringToBool(connection.Parameters.EnableDrive),
		"drive_name":                   connection.Parameters.DriveName,
		"drive_path":                   connection.Parameters.DrivePath,
		"create_drive_path":             stringToBool(connection.Parameters.CreateDrivePath),
		"console":                      stringToBool(connection.Parameters.Console),
		"console_audio":                stringToBool(connection.Parameters.ConsoleAudio),
		"preconnection_id":             connection.Parameters.PreconnectionID,
		"remote_app":                   connection.Parameters.RemoteApp,
		"remote_app_name":              connection.Parameters.RemoteAppName,
		"remote_app_dir":               connection.Parameters.RemoteAppDir,
		"remote_app_args":              connection.Parameters.RemoteAppArgs,
		"load_balance_info":            connection.Parameters.LoadBalanceInfo,
		"disable_copy":                 stringToBool(connection.Parameters.DisableCopy),
		"disable_paste":                stringToBool(connection.Parameters.DisablePaste),
		"read_only":                    stringToBool(connection.Parameters.ReadOnly),
		"resize_method":                connection.Parameters.ResizeMethod,
		"enable_sftp":                  stringToBool(connection.Parameters.EnableSFTP),
		"sftp_hostname":                connection.Parameters.SFTPHostname,
		"sftp_port":                    connection.Parameters.SFTPPort,
		"sftp_username":                connection.Parameters.SFTPUsername,
		"sftp_password":                connection.Parameters.SFTPPassword,
		"sftp_private_key":             connection.Parameters.SFTPPrivateKey,
		"sftp_passphrase":              connection.Parameters.SFTPPassphrase,
		"sftp_host_key":                connection.Parameters.SFTPHostKey,
		"sftp_directory":               connection.Parameters.SFTPDirectory,
		"sftp_root_directory":          connection.Parameters.SFTPRootDirectory,
		"sftp_disable_download":        stringToBool(connection.Parameters.SFTPDisableDownload),
		"sftp_disable_upload":           stringToBool(connection.Parameters.SFTPDisableUpload),
	}
	var parameterList []map[string]interface{}
	parameterList = append(parameterList, parameters)
	d.Set("parameters", parameterList)

	return diags
}

func validateConnectionRDP(d *schema.ResourceData, client *guac.Client) diag.Diagnostics {
	var diags diag.Diagnostics

	attributeDiagnostics := validateStringFields(d.Get("attributes").([]interface{}), []string{
		"guacd_port", "weight", "max_connections", "max_connections_per_user",
	}, restrictedValueAttributes, "attributes")
	if attributeDiagnostics.HasError() {
		diags = append(diags, attributeDiagnostics...)
	}

	parameterDiagnostics := validateStringFields(d.Get("parameters").([]interface{}), []string{
		"port", "width", "height", "dpi", "color_depth", "gateway_port", "sftp_port",
	}, restrictedValueParameters, "parameters")
	if parameterDiagnostics.HasError() {
		diags = append(diags, parameterDiagnostics...)
	}

	for _, key := range []string{"timezone"} {
		timezoneDiagnostics := validateTimezone(d.Get("parameters").([]interface{}), key)
		if timezoneDiagnostics.HasError() {
			diags = append(diags, timezoneDiagnostics...)
		}
	}

	if client == nil {
		return diags
	}
	return diags
}

func convertResourceDataToGuacConnectionRDP(d *schema.ResourceData) (types.GuacConnection, diag.Diagnostics) {
	var connection types.GuacConnection
	var diags diag.Diagnostics

	connection.Name = d.Get("name").(string)
	connection.Identifier = d.Get("identifier").(string)
	connection.ParentIdentifier = d.Get("parent_identifier").(string)
	connection.Protocol = "rdp"

	attributeList := d.Get("attributes").([]interface{})
	if len(attributeList) > 0 {
		attributes := attributeList[0].(map[string]interface{})
		connection.Attributes = types.GuacConnectionAttributes{
			GuacdHostname:         attributes["guacd_hostname"].(string),
			GuacdPort:             attributes["guacd_port"].(string),
			GuacdEncryption:       attributes["guacd_encryption"].(string),
			FailoverOnly:          boolToString(attributes["failover_only"].(bool)),
			Weight:                attributes["weight"].(string),
			MaxConnections:        attributes["max_connections"].(string),
			MaxConnectionsPerUser: attributes["max_connections_per_user"].(string),
		}
	}

	parameterList := d.Get("parameters").([]interface{})
	if len(parameterList) > 0 {
		parameters := parameterList[0].(map[string]interface{})
		connection.Parameters = types.GuacConnectionParameters{
			Hostname:              parameters["hostname"].(string),
			Port:                  parameters["port"].(string),
			Username:              parameters["username"].(string),
			Password:              parameters["password"].(string),
			Domain:                parameters["domain"].(string),
			Security:              parameters["security_mode"].(string),
			DisableAuthentication: boolToString(parameters["disable_authentication"].(bool)),
			IgnoreCert:            boolToString(parameters["ignore_cert"].(bool)),
			GatewayHostname:       parameters["gateway_hostname"].(string),
			GatewayPort:           parameters["gateway_port"].(string),
			GatewayUsername:       parameters["gateway_username"].(string),
			GatewayPassword:       parameters["gateway_password"].(string),
			GatewayDomain:         parameters["gateway_domain"].(string),
			InitialProgram:        parameters["initial_program"].(string),
			ClientName:            parameters["client_name"].(string),
			KeyboardLayout:        parameters["keyboard_layout"].(string),
			Timezone:              parameters["timezone"].(string),
			AdministratorConsole:  boolToString(parameters["administrator_console"].(bool)),
			Width:                 parameters["width"].(string),
			Height:                parameters["height"].(string),
			DPI:                   parameters["dpi"].(string),
			ColorDepth:            parameters["color_depth"].(string),
			DisableAudio:          boolToString(parameters["disable_audio"].(bool)),
			EnableAudioInput:      boolToString(parameters["enable_audio_input"].(bool)),
			EnablePrinting:        boolToString(parameters["enable_printing"].(bool)),
			EnableDrive:           boolToString(parameters["enable_drive"].(bool)),
			DriveName:             parameters["drive_name"].(string),
			DrivePath:             parameters["drive_path"].(string),
			CreateDrivePath:       boolToString(parameters["create_drive_path"].(bool)),
			Console:               boolToString(parameters["console"].(bool)),
			ConsoleAudio:          boolToString(parameters["console_audio"].(bool)),
			PreconnectionID:       parameters["preconnection_id"].(string),
			RemoteApp:             parameters["remote_app"].(string),
			RemoteAppName:         parameters["remote_app_name"].(string),
			RemoteAppDir:          parameters["remote_app_dir"].(string),
			RemoteAppArgs:         parameters["remote_app_args"].(string),
			LoadBalanceInfo:       parameters["load_balance_info"].(string),
			DisableCopy:           boolToString(parameters["disable_copy"].(bool)),
			DisablePaste:          boolToString(parameters["disable_paste"].(bool)),
			ReadOnly:              boolToString(parameters["read_only"].(bool)),
			ResizeMethod:          parameters["resize_method"].(string),
			EnableSFTP:            boolToString(parameters["enable_sftp"].(bool)),
			SFTPHostname:          parameters["sftp_hostname"].(string),
			SFTPPort:              parameters["sftp_port"].(string),
			SFTPUsername:          parameters["sftp_username"].(string),
			SFTPPassword:          parameters["sftp_password"].(string),
			SFTPPrivateKey:        parameters["sftp_private_key"].(string),
			SFTPPassphrase:        parameters["sftp_passphrase"].(string),
			SFTPHostKey:           parameters["sftp_host_key"].(string),
			SFTPDirectory:         parameters["sftp_directory"].(string),
			SFTPRootDirectory:     parameters["sftp_root_directory"].(string),
			SFTPDisableDownload:   boolToString(parameters["sftp_disable_download"].(bool)),
			SFTPDisableUpload:     boolToString(parameters["sftp_disable_upload"].(bool)),
		}
	}

	return connection, diags
}
