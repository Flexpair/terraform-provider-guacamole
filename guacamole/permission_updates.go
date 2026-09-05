package guacamole

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	types "github.com/techBeck03/guacamole-api-client/types"
)

type permissionItemFactory func(string) types.GuacPermissionItem
type permissionApplier func(string, *[]types.GuacPermissionItem) error

type permissionUpdateDefinition struct {
	field    string
	validate func([]string) diag.Diagnostics
	remove   permissionItemFactory
	add      permissionItemFactory
	apply    permissionApplier
}

func stringSetValues(value interface{}) []string {
	set, ok := value.(*schema.Set)
	if !ok || set == nil {
		return nil
	}
	values := make([]string, 0, set.Len())
	for _, item := range set.List() {
		value, ok := item.(string)
		if ok {
			values = append(values, value)
		}
	}
	return values
}

func applyPermissionUpdates(d *schema.ResourceData, identifier string, updates []permissionUpdateDefinition) diag.Diagnostics {
	for _, update := range updates {
		if !d.HasChange(update.field) {
			continue
		}
		if diags := applyPermissionUpdate(d, identifier, update); diags.HasError() {
			return diags
		}
	}
	return nil
}

func applyPermissionUpdate(d *schema.ResourceData, identifier string, update permissionUpdateDefinition) diag.Diagnostics {
	if update.remove == nil || update.add == nil || update.apply == nil {
		return diag.Errorf("incomplete permission update definition for %s", update.field)
	}
	oldValue, newValue := d.GetChange(update.field)
	oldValues := stringSetValues(oldValue)
	newValues := stringSetValues(newValue)
	items := make([]types.GuacPermissionItem, 0, len(oldValues)+len(newValues))

	for _, value := range sliceDiff(oldValues, newValues, false) {
		items = append(items, update.remove(value))
	}

	addedValues := sliceDiff(newValues, oldValues, false)
	if len(addedValues) > 0 && update.validate != nil {
		if diags := update.validate(addedValues); diags.HasError() {
			return diags
		}
	}
	for _, value := range addedValues {
		items = append(items, update.add(value))
	}

	if len(items) == 0 {
		return nil
	}
	if err := update.apply(identifier, &items); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func applyPermissionItems(identifier string, values []string, validate func([]string) diag.Diagnostics, factory permissionItemFactory, apply permissionApplier) diag.Diagnostics {
	if len(values) == 0 {
		return nil
	}
	if factory == nil || apply == nil {
		return diag.Errorf("incomplete permission item handlers for %s", identifier)
	}
	if validate != nil {
		if diags := validate(values); diags.HasError() {
			return diags
		}
	}

	items := make([]types.GuacPermissionItem, 0, len(values))
	for _, value := range values {
		items = append(items, factory(value))
	}
	if err := apply(identifier, &items); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func applyConnectionPermissionItems(identifier string, connections, groups []string, connectionFactory, groupFactory permissionItemFactory, apply permissionApplier) diag.Diagnostics {
	if connectionFactory == nil || groupFactory == nil || apply == nil {
		return diag.Errorf("incomplete connection permission handlers for %s", identifier)
	}
	items := make([]types.GuacPermissionItem, 0, len(connections)+len(groups))
	for _, connection := range connections {
		items = append(items, connectionFactory(connection))
	}
	for _, group := range groups {
		items = append(items, groupFactory(group))
	}
	if len(items) == 0 {
		return nil
	}
	if err := apply(identifier, &items); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
