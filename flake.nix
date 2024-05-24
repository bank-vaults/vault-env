{
  description = "Go libraries for interacting with Hashicorp Vault";

  inputs = {
    # nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    nixpkgs.url = "github:NixOS/nixpkgs/master";
    flake-parts.url = "github:hercules-ci/flake-parts";
    devenv.url = "github:cachix/devenv";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devenv.flakeModule
      ];

      systems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem = { config, self', inputs', pkgs, system, ... }: rec {
        devenv.shells = {
          default = {
            languages = {
              go.enable = true;
              go.package = pkgs.go_1_21;
            };

            services = {
              vault = {
                enable = true;
                package = self'.packages.vault;
              };
            };

            pre-commit.hooks = {
              nixpkgs-fmt.enable = true;
              yamllint.enable = true;
              hadolint.enable = true;
            };

            packages = with pkgs; [
              gnumake

              # golangci-lint
              goreleaser

              kubectl

              yamllint
              hadolint
            ] ++ [
              self'.packages.licensei
              self'.packages.golangci-lint
            ];

            env = {
              KUBECONFIG = "${config.devenv.shells.default.env.DEVENV_STATE}/kube/config";

              VAULT_ADDR = "http://127.0.0.1:8200";
              VAULT_TOKEN = "227e1cce-6bf7-30bb-2d2a-acc854318caf";
            };

            # https://github.com/cachix/devenv/issues/528#issuecomment-1556108767
            containers = pkgs.lib.mkForce { };
          };

          ci = devenv.shells.default;
        };

        packages = {
          # TODO: create flake in source repo
          licensei = pkgs.buildGoModule rec {
            pname = "licensei";
            version = "0.8.0";

            src = pkgs.fetchFromGitHub {
              owner = "goph";
              repo = "licensei";
              rev = "v${version}";
              sha256 = "sha256-Pvjmvfk0zkY2uSyLwAtzWNn5hqKImztkf8S6OhX8XoM=";
            };

            vendorHash = "sha256-ZIpZ2tPLHwfWiBywN00lPI1R7u7lseENIiybL3+9xG8=";

            subPackages = [ "cmd/licensei" ];

            ldflags = [
              "-w"
              "-s"
              "-X main.version=v${version}"
            ];
          };

          vault = pkgs.buildGoModule rec {
            pname = "vault";
            version = "1.14.8";

            src = pkgs.fetchFromGitHub {
              owner = "hashicorp";
              repo = "vault";
              rev = "v${version}";
              sha256 = "sha256-sGCODCBgsxyr96zu9ntPmMM/gHVBBO+oo5+XsdbCK4E=";
            };

            vendorHash = "sha256-zpHjZjgCgf4b2FAJQ22eVgq0YGoVvxGYJ3h/3ZRiyrQ=";

            proxyVendor = true;

            subPackages = [ "." ];

            tags = [ "vault" ];
            ldflags = [
              "-s"
              "-w"
              "-X github.com/hashicorp/vault/sdk/version.GitCommit=${src.rev}"
              "-X github.com/hashicorp/vault/sdk/version.Version=${version}"
              "-X github.com/hashicorp/vault/sdk/version.VersionPrerelease="
            ];
          };

          golangci-lint = pkgs.buildGo121Module rec {
            pname = "golangci-lint";
            version = "1.54.2";

            src = pkgs.fetchFromGitHub {
              owner = "golangci";
              repo = "golangci-lint";
              rev = "v${version}";
              hash = "sha256-7nbgiUrp7S7sXt7uFXX8NHYbIRLZZQcg+18IdwAZBfE=";
            };

            vendorHash = "sha256-IyH5lG2a4zjsg/MUonCUiAgMl4xx8zSflRyzNgk8MR0=";

            subPackages = [ "cmd/golangci-lint" ];

            nativeBuildInputs = [ pkgs.installShellFiles ];

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
              "-X main.commit=v${version}"
              "-X main.date=19700101-00:00:00"
            ];

            postInstall = ''
              for shell in bash zsh fish; do
                HOME=$TMPDIR $out/bin/golangci-lint completion $shell > golangci-lint.$shell
                installShellCompletion golangci-lint.$shell
              done
            '';

            meta = with pkgs.lib; {
              description = "Fast linters Runner for Go";
              homepage = "https://golangci-lint.run/";
              changelog = "https://github.com/golangci/golangci-lint/blob/v${version}/CHANGELOG.md";
              license = licenses.gpl3Plus;
              maintainers = with maintainers; [ anpryl manveru mic92 ];
            };
          };
        };
      };
    };
}
