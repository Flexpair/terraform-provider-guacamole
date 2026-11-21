package guacamole

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	types "github.com/techBeck03/guacamole-api-client/types"
)

// connectionValidationSpec groups the static validation rules for one
// connection protocol. The same spec drives plan-time schema validation
// (applyConnectionValidation) and the apply-time checks run by Create/Update
// (validateConnectionSpec), so plan and apply behavior stay in sync.
type connectionValidationSpec struct {
	attributeIntegers    []string
	restrictedAttributes map[string][]string
	parameterIntegers    []string
	restrictedParameters map[string][]string
	timezoneParameter    bool
}

func rdpValidationSpec() connectionValidationSpec {
	var attributeInterface types.GuacConnectionAttributes
	var parameterInterface types.GuacConnectionParameters
	return connectionValidationSpec{
		attributeIntegers: []string{
			"guacd_port", "weight", "max_connections", "max_connections_per_user",
		},
		restrictedAttributes: map[string][]string{
			"guacd_encryption": attributeInterface.ValidEncryptionTypes(),
		},
		parameterIntegers: []string{
			"port", "gateway_port", "width", "height", "dpi", "preconnection_id",
			"sftp_port", "sftp_keepalive_interval", "wol_boot_wait_time",
		},
		restrictedParameters: map[string][]string{
			"security_mode":   parameterInterface.ValidSecurityModes(),
			"keyboard_layout": parameterInterface.ValidKeyboardLayouts(),
			"color_depth":     parameterInterface.ValidColorDepths(),
			"resize_method":   parameterInterface.ValidResizeMethods(),
		},
		timezoneParameter: true,
	}
}

func sshValidationSpec() connectionValidationSpec {
	var attributeInterface types.GuacConnectionAttributes
	var parameterInterface types.GuacConnectionParameters
	return connectionValidationSpec{
		attributeIntegers: []string{
			"guacd_port", "weight", "max_connections", "max_connections_per_user",
		},
		restrictedAttributes: map[string][]string{
			"guacd_encryption": attributeInterface.ValidEncryptionTypes(),
		},
		parameterIntegers: []string{
			"port", "max_scrollback_size", "server_keepalive", "wol_boot_wait_time",
		},
		restrictedParameters: map[string][]string{
			"color_scheme":  parameterInterface.ValidColorSchemes(),
			"font_size":     parameterInterface.ValidFontSizes(),
			"backspace":     parameterInterface.ValidBackspaceCodes(),
			"terminal_type": parameterInterface.ValidTerminalTypes(),
		},
		timezoneParameter: true,
	}
}

func telnetValidationSpec() connectionValidationSpec {
	var attributeInterface types.GuacConnectionAttributes
	var parameterInterface types.GuacConnectionParameters
	return connectionValidationSpec{
		attributeIntegers: []string{
			"guacd_port", "weight", "max_connections", "max_connections_per_user",
		},
		restrictedAttributes: map[string][]string{
			"guacd_encryption": attributeInterface.ValidEncryptionTypes(),
		},
		parameterIntegers: []string{
			"port", "wol_boot_wait_time",
		},
		restrictedParameters: map[string][]string{
			"color_scheme":  parameterInterface.ValidColorSchemes(),
			"font_size":     parameterInterface.ValidFontSizes(),
			"backspace":     parameterInterface.ValidBackspaceCodes(),
			"terminal_type": parameterInterface.ValidTerminalTypes(),
		},
	}
}

func vncValidationSpec() connectionValidationSpec {
	var attributeInterface types.GuacConnectionAttributes
	var parameterInterface types.GuacConnectionParameters
	return connectionValidationSpec{
		attributeIntegers: []string{
			"guacd_port", "weight", "max_connections", "max_connections_per_user",
		},
		restrictedAttributes: map[string][]string{
			"guacd_encryption": attributeInterface.ValidEncryptionTypes(),
		},
		parameterIntegers: []string{
			"port", "destination_port", "sftp_port", "sftp_keepalive_interval", "wol_boot_wait_time",
		},
		restrictedParameters: map[string][]string{
			"cursor":             parameterInterface.ValidCursors(),
			"color_depth":        parameterInterface.ValidColorDepths(),
			"clipboard_encoding": parameterInterface.ValidClipboardEncodings(),
		},
	}
}

func kubernetesValidationSpec() connectionValidationSpec {
	var attributeInterface types.GuacConnectionAttributes
	var parameterInterface types.GuacConnectionParameters
	return connectionValidationSpec{
		attributeIntegers: []string{
			"guacd_port", "weight", "max_connections", "max_connections_per_user",
		},
		restrictedAttributes: map[string][]string{
			"guacd_encryption": attributeInterface.ValidEncryptionTypes(),
		},
		parameterIntegers: []string{
			"port", "max_scrollback_size",
		},
		restrictedParameters: map[string][]string{
			"color_scheme": parameterInterface.ValidColorSchemes(),
			"font_size":    parameterInterface.ValidFontSizes(),
			"backspace":    parameterInterface.ValidBackspaceCodes(),
		},
	}
}

// validateConnectionSpec runs the apply-time validation for a connection
// resource using its protocol spec. Empty values are skipped, mirroring the
// plan-time validators.
func validateConnectionSpec(d *schema.ResourceData, spec connectionValidationSpec) diag.Diagnostics {
	var diags diag.Diagnostics
	attributeList := d.Get("attributes").([]interface{})
	diags = append(diags, validateStringFields(attributeList, spec.attributeIntegers, spec.restrictedAttributes, "attribute")...)
	parameterList := d.Get("parameters").([]interface{})
	diags = append(diags, validateStringFields(parameterList, spec.parameterIntegers, spec.restrictedParameters, "parameter")...)
	if spec.timezoneParameter {
		diags = append(diags, validateTimezone(parameterList, "timezone")...)
	}
	return diags
}

func validateConnectionRDP(d *schema.ResourceData) diag.Diagnostics {
	return validateConnectionSpec(d, rdpValidationSpec())
}

func validateConnectionSSH(d *schema.ResourceData) diag.Diagnostics {
	return validateConnectionSpec(d, sshValidationSpec())
}

func validateConnectionTelnet(d *schema.ResourceData) diag.Diagnostics {
	return validateConnectionSpec(d, telnetValidationSpec())
}

func validateConnectionVNC(d *schema.ResourceData) diag.Diagnostics {
	return validateConnectionSpec(d, vncValidationSpec())
}

func validateConnectionKubernetes(d *schema.ResourceData) diag.Diagnostics {
	return validateConnectionSpec(d, kubernetesValidationSpec())
}

func validateConnectionGroup(d *schema.ResourceData) diag.Diagnostics {
	var diags diag.Diagnostics

	connectionGroupInterface := types.GuacConnectionGroup{}
	// validate connection group restricted values
	connectionGroupRestrictedValueParameters := map[string][]string{
		"type": connectionGroupInterface.ValidTypes(),
	}
	// parameters that need to be forced to upper case
	forceUpperCaseParameters := []string{
		"type",
	}

	for k, v := range connectionGroupRestrictedValueParameters {
		var value string
		capitalizationCheck := stringInSlice(forceUpperCaseParameters, []string{k})
		if !capitalizationCheck.HasError() {
			value = strings.ToUpper(d.Get(k).(string))
		} else {
			value = d.Get(k).(string)
		}
		check := stringInSlice(v, []string{value})
		if check.HasError() {
			diags = append(diags, check...)
		}
	}

	// validate attributes
	attributeList := d.Get("attributes").([]interface{})

	stringIntAttributes := []string{
		"max_connections",
		"max_connections_per_user",
	}

	if len(attributeList) > 0 {
		attributes := attributeList[0].(map[string]interface{})
		// validate string integer values
		for _, v := range stringIntAttributes {
			if attributes[v].(string) != "" {
				_, err := strconv.Atoi(attributes[v].(string))
				if err != nil {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Invalid entry",
						Detail:   fmt.Sprintf("Expected string integer for attribute key: %s but was unable to convert: %s to integer", v, attributes[v].(string)),
					})
				}
			}
		}
	}

	return diags
}

// validateIntegerString is a plan-time validator for fields that model
// integers as strings (Guacamole's parameter values are all strings).
func validateIntegerString(value interface{}, key string) ([]string, []error) {
	stringValue, ok := value.(string)
	if !ok || stringValue == "" {
		return nil, nil
	}
	if _, err := strconv.Atoi(stringValue); err != nil {
		return nil, []error{fmt.Errorf("expected an integer string for %s but got: %s", key, stringValue)}
	}
	return nil, nil
}

// validateRestrictedStringValue is a plan-time validator for fields restricted
// to a fixed set of values. Empty values are allowed; Guacamole applies its
// own defaults.
func validateRestrictedStringValue(valid []string) func(interface{}, string) ([]string, []error) {
	return func(value interface{}, key string) ([]string, []error) {
		stringValue, ok := value.(string)
		if !ok || stringValue == "" {
			return nil, nil
		}
		return validation.StringInSlice(valid, false)(value, key)
	}
}

// validateTimezoneString is a plan-time validator for timezone parameters.
func validateTimezoneString(value interface{}, key string) ([]string, []error) {
	stringValue, ok := value.(string)
	if !ok || stringValue == "" {
		return nil, nil
	}
	if _, err := time.LoadLocation(stringValue); err != nil {
		return nil, []error{fmt.Errorf("unable to process timezone string: %s", stringValue)}
	}
	return nil, nil
}

// applyConnectionValidation attaches plan-time schema validators derived from
// the protocol spec to the nested attributes and parameters blocks.
func applyConnectionValidation(resource *schema.Resource, spec connectionValidationSpec) *schema.Resource {
	applyToBlock := func(blockName string, integers []string, restricted map[string][]string, timezone bool) {
		blockField, ok := resource.Schema[blockName]
		if !ok {
			return
		}
		block, ok := blockField.Elem.(*schema.Resource)
		if !ok {
			return
		}
		for _, key := range integers {
			if field, ok := block.Schema[key]; ok {
				field.ValidateDiagFunc = validation.ToDiagFunc(validateIntegerString)
			}
		}
		for key, validValues := range restricted {
			if field, ok := block.Schema[key]; ok {
				field.ValidateDiagFunc = validation.ToDiagFunc(validateRestrictedStringValue(validValues))
			}
		}
		if timezone {
			if field, ok := block.Schema["timezone"]; ok {
				field.ValidateDiagFunc = validation.ToDiagFunc(validateTimezoneString)
			}
		}
	}
	applyToBlock("attributes", spec.attributeIntegers, spec.restrictedAttributes, false)
	applyToBlock("parameters", spec.parameterIntegers, spec.restrictedParameters, spec.timezoneParameter)
	return resource
}

// applyConnectionGroupValidation attaches plan-time schema validators for the
// connection group resource (type enumeration and integer-like attributes).
func applyConnectionGroupValidation(resource *schema.Resource) *schema.Resource {
	connectionGroupInterface := types.GuacConnectionGroup{}
	if typeField, ok := resource.Schema["type"]; ok {
		typeField.ValidateDiagFunc = validation.ToDiagFunc(
			validation.StringInSlice(connectionGroupInterface.ValidTypes(), true),
		)
	}
	attributesField, ok := resource.Schema["attributes"]
	if !ok {
		return resource
	}
	block, ok := attributesField.Elem.(*schema.Resource)
	if !ok {
		return resource
	}
	for _, key := range []string{"max_connections", "max_connections_per_user"} {
		if field, ok := block.Schema[key]; ok {
			field.ValidateDiagFunc = validation.ToDiagFunc(validateIntegerString)
		}
	}
	return resource
}
