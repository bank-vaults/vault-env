# `vault-env`

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/bank-vaults/vault-env/ci.yaml?branch=main&style=flat-square)](https://github.com/bank-vaults/vault-env/actions/workflows/ci.yaml?query=workflow%3ACI)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/bank-vaults/vault-env/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/bank-vaults/vault-env)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/8062/badge)](https://www.bestpractices.dev/projects/8062)

**Minimalistic init system for containers with [Hashicorp Vault](https://www.vaultproject.io/) secrets support .**

## Usage

`vault-env` is designed for use with the [Kubernetes mutating webhook](https://bank-vaults.dev/docs/mutating-webhook/); however, it can also function as a standalone tool.

## Development

Install [Go](https://go.dev/dl/) on your computer then run `make deps` to install the rest of the dependencies.

Make sure Docker is installed with Compose and Buildx.

Run project dependencies:

```shell
make up
```

Build a binary:

```shell
make build
```

Run the test suite:

```shell
make test
```

Run linters:

```shell
make lint # pass -j option to run them in parallel
```

Some linter violations can automatically be fixed:

```shell
make fmt
```

Build artifacts locally:

```shell
make artifacts
```

Once you are done either stop or tear down dependencies:

```shell
make down
```

## License

The project is licensed under the [Apache 2.0 License](LICENSE).
