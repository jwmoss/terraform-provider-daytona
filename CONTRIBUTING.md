# Contributing

Thanks for your interest in improving the Terraform provider for Daytona.

## Requirements

- Go 1.25 or newer
- Terraform 1.0 or newer; provider-defined actions require Terraform 1.14 or newer
- [`prek`](https://github.com/j178/prek) for the local git hooks

## Build

```shell
make build      # compile the provider binary
make install    # build and install into $GOPATH/bin
```

## Test

Unit tests run without any Daytona credentials:

```shell
make test
```

Acceptance tests create real Daytona resources and require credentials. They are
gated behind `TF_ACC=1` and, for the opt-in suites, additional environment
variables. See the [Development section of the README](README.md#development) for
the full set of acceptance commands and the credentials each one needs.

```shell
make testacc
```

If an acceptance run is interrupted and leaves resources behind, clean up
everything created with the `tf-acc-` name prefix:

```shell
DAYTONA_API_KEY="dtn_..." go test ./internal/provider/ -sweep=all
```

## Lint and format

```shell
make lint   # golangci-lint
make fmt    # gofmt
```

Install and run the git hooks before opening a pull request:

```shell
prek install
prek run --all-files
```

## Documentation

Provider documentation under `docs/` is generated from the schema and the
examples in `examples/`. Do not edit `docs/` by hand. After changing schemas or
examples, regenerate and commit the result:

```shell
make generate
```

CI fails if generated documentation is out of date, so run this whenever you
touch a resource, data source, action, or its example.

## Pull requests

- Describe only what is in the diff; no discarded approaches or prior iterations.
- One logical change per pull request.
- Use [Conventional Commits](https://www.conventionalcommits.org/) for commit
  messages, imperative mood, subject under 72 characters.
- Fill out the pull request template, including the commands or live acceptance
  checks you ran.
- Note any API compatibility, state migration, or sensitive-state
  considerations.

## Reporting issues

Use the issue templates for bug reports and feature requests. For security
vulnerabilities, follow [SECURITY.md](SECURITY.md) instead of opening a public
issue.
