# Examples

This directory contains examples that are mostly used for documentation, but can also be run/tested manually via the Terraform CLI.

The document generation tool looks for files in the following locations by default. All other *.tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating examples that can run and/or are testable even if some parts are not relevant for the documentation.

* **provider/provider.tf** example file for the provider index page
* **data-sources/`full data source name`/data-source.tf** example file for the named data source page
* **resources/`full resource name`/resource.tf** example file for the named data source page

Additional runnable examples live outside the documentation generator paths:

* **modules/daytona-sandbox** reusable module for provisioning a Daytona sandbox and optional persistent volume
* **github-source-module** root example that consumes the module from this public GitHub repository
* **use-cases** complete configurations for the platform-infrastructure side of Daytona (region/runner registration, organization governance, golden snapshots, service API keys); resources and data sources only, so they also work with OpenTofu
