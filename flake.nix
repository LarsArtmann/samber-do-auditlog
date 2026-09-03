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
          goPkg = pkgs.go; # bootstrap only; GOTOOLCHAIN below pins the exact 1.23 toolchain
        in
        {
          devShells.default = pkgs.mkShellNoCC {
            packages = builtins.attrValues {
              inherit (pkgs)
                go # bootstrap; toolchain pinned via GOTOOLCHAIN
                golangci-lint
                actionlint
                govulncheck
                golines
                nixfmt
                ;
            };

            GOTOOLCHAIN = "go1.23.12";
            # Explicitly clear the experiment: a stale GOEXPERIMENT=jsonv2
            # from an ambient shell otherwise breaks the go1.23 toolchain
            # ("unknown GOEXPERIMENT jsonv2").
            GOEXPERIMENT = "";
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
              program = "${
                pkgs.writeShellApplication {
                  name = "coverage-gate";
                  runtimeInputs = [
                    goPkg
                    pkgs.stdenv.cc
                  ];
                  text = ''
                    # -race requires cgo; the C toolchain must be on PATH.
                    export CGO_ENABLED=1
                    export GOTOOLCHAIN=go1.23.12
                    exec sh ./scripts/coverage-gate.sh "$@"
                  '';
                }
              }/bin/coverage-gate";
            };

            auditlog = {
              type = "app";
              program = "${
                pkgs.writeShellApplication {
                  name = "auditlog";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    export CGO_ENABLED=0
                    export GOTOOLCHAIN=go1.23.12
                    exec go run ./cmd/auditlog "$@"
                  '';
                }
              }/bin/auditlog";
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
