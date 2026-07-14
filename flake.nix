{
  description = "Audit logging plugin for samber/do v2";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    systems.url = "github:nix-systems/default";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          goPkg = pkgs.go_1_26;
        in
        {
          devShells.default = pkgs.mkShellNoCC {
            packages = builtins.attrValues {
              inherit (pkgs)
                go_1_26
                golangci-lint
                actionlint
                govulncheck
                golines
                nixfmt
                ;
            };

            BUILDFLOW_LANGUAGE = "go";
          };

          packages.default =
            pkgs.runCommand "samber-do-auditlog"
              {
                meta = with lib; {
                  description = "Audit logging plugin for samber/do v2";
                  homepage = "https://github.com/larsartmann/samber-do-auditlog";
                  license = licenses.mit;
                  platforms = platforms.unix;
                };
              }
              ''
                mkdir -p $out
              '';

          apps = {
            coverage = {
              type = "app";
              program = toString (
                pkgs.writeShellApplication {
                  name = "coverage-gate";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    export CGO_ENABLED=0
                    exec sh ./scripts/coverage-gate.sh "$@"
                  '';
                }
              );
            };

            auditlog = {
              type = "app";
              program = toString (
                pkgs.writeShellApplication {
                  name = "auditlog";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    export CGO_ENABLED=0
                    exec go run ./cmd/auditlog "$@"
                  '';
                }
              );
            };

            default = config.apps.auditlog;
          };

          treefmt = {
            programs = {
              nixfmt.enable = true;
              gofmt.enable = true;
            };
          };

          checks.build = config.packages.default;
          checks.format = config.treefmt.build.check self;
        };
    };
}
