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
          goPkg = pkgs.go_1_27;
        in
        {
          devShells.default = pkgs.mkShellNoCC {
            packages = builtins.attrValues {
              inherit (pkgs)
                go_1_27
                golangci-lint
                actionlint
                govulncheck
                golines
                nixfmt
                ;
            };

            GOEXPERIMENT = "jsonv2";
            # go_1_27 IS 1.27.1; `local` stops go from downloading the
            # exact-version toolchain (GOTOOLCHAIN=go1.27.1 would fetch it
            # whenever the ambient go differs — sandbox DNS death).
            GOTOOLCHAIN = "local";
            BUILDFLOW_LANGUAGE = "go";
          };

          # Real package: the auditlog CLI (cmd/auditlog). Replaces a former
          # `mkdir -p $out` marker whose checks.build asserted nothing.
          packages.default = pkgs.buildGo127Module {
            pname = "auditlog";
            version = self.shortRev or self.dirtyShortRev or "dev";
            src = lib.fileset.toSource {
              root = ./.;
              fileset = lib.fileset.gitTracked ./.;
            };
            subPackages = [ "cmd/auditlog" ];
            vendorHash = "sha256-1vNoG2t5V1KEFMhWZiMza9OHCHoUl68Mw1KWLRTfC1k=";
            meta = with lib; {
              description = "Audit logging CLI for samber/do v2";
              homepage = "https://github.com/larsartmann/samber-do-auditlog";
              license = licenses.mit;
              platforms = platforms.unix;
              mainProgram = "auditlog";
            };
          };

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
                    export GOTOOLCHAIN=local
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
                    export GOTOOLCHAIN=local
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
