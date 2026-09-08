package guacamole

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func sensitiveOptionalComputedString(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: description,
		Optional:    true,
		Computed:    true,
		Sensitive:   true,
	}
}

func sensitiveComputedString(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: description,
		Computed:    true,
		Sensitive:   true,
	}
}
