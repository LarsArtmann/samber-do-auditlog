{
  description = "DevShell for samber-do-auditlog — Go 1.26.3, golangci-lint, govulncheck";

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

            # HTML template code generation is handled by Go's `tool` directive
            # (see go.mod). No external templ binary needed — `go tool templ`
            # builds the exact go.mod-pinned version automatically.

            # Code formatting.
            golines
            nixpkgs-fmt
          ];
        };

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
