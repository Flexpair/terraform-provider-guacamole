Thank you for opening an issue. This tracker is reserved for bug reports and feature requests about this provider.

### Terraform Version

Run `terraform -v` to show the version. If you are not running the latest version of Terraform, please upgrade because your issue may have already been fixed.

### Provider Version

Which release of this fork (see the version history in `README.md`), or the commit SHA.

### Affected Resource(s)

Please list the affected resources as a list, for example:

- guacamole_connection_ssh
- guacamole_user

### Terraform Configuration Files

```hcl
# Copy-paste a minimal configuration. Redact credentials, tokens, and
# private keys before pasting.
```

### Debug Output

Please provide a link to a GitHub Gist containing the debug output (`TF_LOG=DEBUG`). Redact credentials and tokens. Do NOT paste the debug output directly into the issue.

### Panic Output

If Terraform produced a panic, please provide a link to a GitHub Gist containing the output of the `crash.log`.

### Expected Behavior

What should have happened?

### Actual Behavior

What actually happened?

### Steps to Reproduce

Please provide the smallest set of steps to reproduce the issue, for example:

1. `terraform plan`
2. `terraform apply`
