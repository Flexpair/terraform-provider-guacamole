---
page_title: "User Resource - terraform-provider-guacamole"
subcategory: ""
description: |-
  The user resource allows you to configure a guacamole user
---

# Resource `guacamole_user`

The user resource allows you to configure a guacamole user

## Example Usage

```terraform
resource "guacamole_user" "user" {
  username = "testGuacamoleUser"
  password = "password"
  attributes {
    full_name = "Test User"
    email = "testUser@example.com"
    timezone = "America/Chicago"
  }
  system_permissions = ["ADMINISTER", "CREATE_USER"]
  group_membership = ["Parent Group"]
  connections = [
    "12345"
  ]
  connection_groups = [
    "678910"
  ]
}

```

## Argument Reference

### Base

- `username` -  (string, Required) the guacamole user username
- `password` -  (string, Sensitive) the guacamole user password (write-only: the API never returns it, so the configured value is preserved in state)
- `group_membership` - (Set) identifiers of the user groups this user belongs to
- `system_permissions` - (Set) system permissions assigned to the user
- `connections` - (Set) connection identifiers assigned to the user.  This list currently does not include connection identifiers from parent user groups.
- `connection_groups` - (Set) connection group identifiers assigned to the user.  This list currently does not include connection group identifiers from parent user groups.

> **Note:** `group_membership`, `system_permissions`, `connections`, and `connection_groups` are
> authoritative sets. Permissions or memberships granted outside Terraform (for example, manually
> in the Guacamole UI) are removed on the next apply.

### Attributes

- `organizational_role` - (string) assigned organizational role
- `full_name` - (string) full name
- `email` - (string) email address
- `expired` - (bool) whether the account is expired
- `timezone` - (string) the timezone string ("America/Chicago")
- `access_window_start` - (string) access window start time
- `access_window_end` - (string) access window end time
- `disabled` - (bool) whether the account is disabled
- `valid_from` - (string) account valid start date
- `valid_until` - (string) account valid end date

## Attributes Reference

In addition to all the arguments above, the following attributes are exported.

#### Base
- `last_active` - (string) timestamp of last activity

## Import

User can be imported using the `resource id`, e.g.

```shell
terraform import guacamole_user.user user_name
```
