package guacamole

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func stringToBool(v string) bool {
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

func boolToString(b bool) string {
	if b == true {
		return "true"
	}
	return ""
}

func validateStringFields(values []interface{}, integerKeys []string, restrictedFields map[string][]string, fieldKind string) diag.Diagnostics {
	if len(values) == 0 {
		return nil
	}

	fields := values[0].(map[string]interface{})
	return validateStringFieldMap(fields, integerKeys, restrictedFields, fieldKind)
}

func validateStringFieldMap(fields map[string]interface{}, integerKeys []string, restrictedFields map[string][]string, fieldKind string) diag.Diagnostics {
	diags := validateStringIntegers(fields, integerKeys, fieldKind)
	return append(diags, validateRestrictedStrings(fields, restrictedFields)...)
}

func validateStringIntegers(fields map[string]interface{}, keys []string, fieldKind string) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range keys {
		value := fields[key].(string)
		if value == "" {
			continue
		}
		if _, err := strconv.Atoi(value); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid entry",
				Detail:   fmt.Sprintf("Expected string integer for %s key: %s but was unable to convert: %s to integer", fieldKind, key, value),
			})
		}
	}
	return diags
}

func validateRestrictedStrings(fields map[string]interface{}, restrictedFields map[string][]string) diag.Diagnostics {
	var diags diag.Diagnostics
	for key, validValues := range restrictedFields {
		value := fields[key].(string)
		if value != "" {
			diags = append(diags, stringInSlice(validValues, []string{value})...)
		}
	}
	return diags
}

func validateTimezone(values []interface{}, key string) diag.Diagnostics {
	if len(values) == 0 {
		return nil
	}

	timezone := values[0].(map[string]interface{})[key].(string)
	if _, err := time.LoadLocation(timezone); err == nil {
		return nil
	}

	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  "Invalid timezone",
		Detail:   fmt.Sprintf("Unable to process timezone string: %s", timezone),
	}}
}

func sliceDiff(slice1 []string, slice2 []string, bidirectional bool) []string {
	var diff []string

	var loopCount int
	if bidirectional {
		loopCount = 2
	} else {
		loopCount = 1
	}

	for i := 0; i < loopCount; i++ {
		for _, v1 := range slice1 {
			match := false
			for _, v2 := range slice2 {
				if v1 == v2 {
					match = true
				}
			}
			if !match {
				diff = append(diff, v1)
			}
		}
		if bidirectional {
			slice1, slice2 = slice2, slice1
		}
	}
	return diff
}

func stringInSlice(valid, test []string) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, t := range test {
		match := false
		for _, v := range valid {
			if t == v {
				match = true
			}
		}
		if !match {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid entry",
				Detail:   fmt.Sprintf("%s is not a valid entry", t),
			})
		}
	}
	return diags
}

func checkForDuplicates(slice1 []string) diag.Diagnostics {
	var diags diag.Diagnostics
	var output []string
	for _, v1 := range slice1 {
		if !stringInSlice(output, []string{v1}).HasError() {
			output = append(output, v1)
		} else {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Duplicate entry",
				Detail:   fmt.Sprintf("%s is duplicated", v1),
			})
		}
	}
	return diags
}

// sorts slice 2 by slice 1
func sortSliceBySlice(slice1, slice2 []string) []string {
	var sorted []string
	for _, v1 := range slice1 {
		for _, v2 := range slice2 {
			if v1 == v2 {
				sorted = append(sorted, v2)
			}
		}
	}
	return sorted
}

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func convertStringSliceToInterfaceSlice(slice []string) []interface{} {
	var output []interface{}
	for _, v := range slice {
		output = append(output, v)
	}
	return output
}

func convertInterfaceSliceToStringSlice(slice []interface{}) []string {
	var output []string
	for _, v := range slice {
		output = append(output, v.(string))
	}
	return output
}

func listToSet(list []string) *schema.Set {
	return schema.NewSet(schema.HashString, convertStringSliceToInterfaceSlice(list))
}

func stringToTime(input string) time.Time {
	if input == "" {
		return time.Time{}
	}
	output, _ := time.Parse("2006-01-02", input)
	return output
}

func boolToTerraformString(b bool) string {
	return strconv.FormatBool(b)
}

func stringToTerraformBool(s string) bool {
	output, _ := strconv.ParseBool(s)
	return output
}

func stringSliceToTerraformSet(d *schema.ResourceData, key string, values []string) error {
	return d.Set(key, values)
}

func primitiveToHclString(value interface{}, isNested bool) string {
	var output string
	if isNested {
		output = "{"
	}
	switch value := value.(type) {
	case string:
		output += fmt.Sprintf("\"%s\"", value)
	case bool:
		output += fmt.Sprintf("%t", value)
	case int:
		output += fmt.Sprintf("%d", value)
	case []interface{}:
		output += "["
		for _, v := range value {
			output += primitiveToHclString(v, false)
		}
		output += "]"
	case map[string]interface{}:
		for k, v := range value {
			output += fmt.Sprintf("%s = %s", k, primitiveToHclString(v, true))
		}
	}
	if isNested {
		output += "}"
	}
	return output
}

func validateTimestring(timeString, name string) diag.Diagnostics {
	var diags diag.Diagnostics
	regex := `^\d{4}[-]\d{2}[-]\d{2}$`
	matched, err := regexp.MatchString(regex, timeString)
	if err != nil || !matched {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Invalid entry",
			Detail:   fmt.Sprintf("%s must be a valid date formatted as YYYY-MM-DD", name),
		})
	}
	return diags
}

func toHclString(data map[string]interface{}, isNested bool) string {
	var output string
	if isNested {
		output = "{"
	}
	for k, v := range data {
		output += fmt.Sprintf("%s = %s", k, primitiveToHclString(v, true))
	}
	if isNested {
		output += "}"
	}
	return output
}

func testAccCheckTestSliceVals(n string, key string, expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		for _, value := range expected {
			if !contains(resource.Primary.Attributes[key], value) {
				return fmt.Errorf("%s does not contain expected value: %s", key, value)
			}
		}
		return nil
	}
}

func readResourceDataTestAttr(resource *terraform.Resource, key string) string {
	return resource.Primary.Attributes[key]
}
