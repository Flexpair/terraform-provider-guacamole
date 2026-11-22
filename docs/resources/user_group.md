---
page_title: "User Group Resource - terraform-provider-guacamole"
subcategory: ""
description: |-
  The user group resource allows you to configure a guacamole user group by identifier
---

# Resource `guacamole_user_group`

The user group resource allows you to configure a guacamole user group

## Example Usage

```terraform
resource "guacamole_user_group" "group" {
  identifier = "testGuacamoleUserGroup"
  system_permissions = ["ADMINISTER", "CREATE_USER"]
  group_membership = ["Parent Group"]
  connections = [
    "12345"
  ]
  connection_groups = [
    "678910"
  ]
  attributes {
    disabled = true
  }
}

```

## Argument Reference

### Base

- `identifier` -  (string, Required, ForceNew) the guacamole user group identifier
- `group_membership` - (Set) identifiers of the parent user groups this group is a member of
- `system_permissions` - (Set) system permissions assigned to the user group
- `connections` - (Set) connection identifiers assigned to the user group.  This list currently does not include connection identifiers from parent user groups.
- `connection_groups` - (Set) connection group identifiers assigned to the user group.  This list currently does not include connection group identifiers from parent user groups.

> **Note:** `group_membership`, `system_permissions`, `connections`, and `connection_groups` are
> authoritative sets. Permissions or memberships granted outside Terraform (for example, manually
> in the Guacamole UI) are removed on the next apply.
>
> **Behavior change:** `group_membership` now identifies parent groups this group belongs to,
> rather than child groups that belong to this group. Update existing configurations that list
> child groups; otherwise, the listed groups will become parents and inherited permissions will
> flow in the opposite direction.

### Attributes

- `attributes` - (Block List, Max: 1) attributes of the user group:
  - `disabled` - (bool) whether the group is disabled

## Import

User group can be imported using the `resource id`, e.g.

```shell
terraform import guacamole_user_group.group group_name
```
