# `vault-env`

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/bank-vaults/vault-env/ci.yaml?branch=main&style=flat-square)](https://github.com/bank-vaults/vault-env/actions/workflows/ci.yaml?query=workflow%3ACI)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/bank-vaults/vault-env/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/bank-vaults/vault-env)

**Minimalistic init system for containers with [Hashicorp Vault](https://www.vaultproject.io/) secrets support .**

## Usage

`vault-env` is designed for use with the [Kubernetes mutating webhook](https://bank-vaults.dev/docs/mutating-webhook/); however, it can also function as a standalone tool.

## Development

**For an optimal developer experience, it is recommended to install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html).**

_Alternatively, install [Go](https://go.dev/dl/) on your computer then run `make deps` to install the rest of the dependencies._

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

Run the linter:

```shell
make lint
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
make stop

# OR

make down
```

## License

The project is licensed under the [Apache 2.0 License](LICENSE).
