{
  description = "DevShell for samber-do-auditlog — Go 1.26.3, golangci-lint, govulncheck";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, flake-utils, self, ... }:
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

            # GitHub Actions workflow validation.
            actionlint

            # Vulnerability scanning.
            govulncheck

            # HTML template code generation is handled by Go's `tool` directive
            # (see go.mod). No external templ binary needed — `go tool templ`
            # builds the exact go.mod-pinned version automatically.

            # Code formatting.
            golines
            nixpkgs-fmt
          ];

          # buildflow auto-detects "nix" when flake.nix is present; force Go.
          BUILDFLOW_LANGUAGE = "go";
        };

        # This is a Go library — there is no buildable binary. The package
        # output provides metadata so `nix build` succeeds for tooling that
        # expects a default derivation (e.g. buildflow's nix-build step).
        packages.default = pkgs.runCommand "samber-do-auditlog"
          {
            meta = with pkgs.lib; {
              description = "Audit logging plugin for samber/do v2";
              homepage = "https://github.com/larsartmann/samber-do-auditlog";
              license = licenses.mit;
              mainProgram = "samber-do-auditlog";
              platforms = platforms.unix;
            };
          } ''
          mkdir -p $out
          cat > $out/README <<EOF
          samber-do-auditlog — audit logging plugin for samber/do v2.
          This is a library; use devShells.default for development.
          EOF
        '';

        # Runnable apps. Invoke with: nix run .#<name>
        # They wrap go so no vendorHash is required; run from the repo root.
        apps.coverage = {
          type = "app";
          program = toString (pkgs.writeShellScript "coverage-gate" ''
            export PATH="${pkgs.lib.makeBinPath [ pkgs.go_1_26 ]}"
            export CGO_ENABLED=0
            exec sh ./scripts/coverage-gate.sh "$@"
          '');
        };

        apps.auditlog = {
          type = "app";
          program = toString (pkgs.writeShellScript "auditlog" ''
            export PATH="${pkgs.lib.makeBinPath [ pkgs.go_1_26 ]}"
            export CGO_ENABLED=0
            exec go run ./cmd/auditlog "$@"
          '');
        };

        apps.default = self.apps.${system}.auditlog;

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
