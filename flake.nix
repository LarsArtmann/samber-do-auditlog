{
  description = "DevShell for samber-do-auditlog — Go 1.26.3, templ, golangci-lint, govulncheck";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Core toolchain — pinned to match go.mod (Go 1.26.3).
            go_1_26

            # Linting and analysis.
            golangci-lint

            # Vulnerability scanning.
            govulncheck

            # HTML template code generation.
            # NOTE: nixpkgs templ may differ from the go.mod pin (v0.3.1020).
            # CI's stale-generation check uses `go install ...@v0.3.1020`
            # as the authoritative version. If local regeneration produces
            # drift, install the exact version:
            #   go install github.com/a-h/templ/cmd/templ@v0.3.1020
            templ

            # Code formatting.
            golines
            nixpkgs-fmt
          ];
        };
      }
    );
}
