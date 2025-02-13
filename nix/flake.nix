# ref: https://github.com/hercules-ci/flake-parts
# ref: https://ryantm.github.io/nixpkgs/builders/special/mkshell/
# ref: https://github.com/kamadorueda/alejandra
{
  outputs = inputs @ {
    nixpkgs,
    flake-parts,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      # Re-use nix-package system set:
      systems = builtins.attrNames nixpkgs.legacyPackages;

      # Define flake outputs:
      perSystem = {
        config,
        system,
        pkgs,
        ...
      }: {
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.alejandra # fmt: for nix configs

            # Standard env dependencies
            pkgs.which
            pkgs.git

            # Developer dependencies
            pkgs.go
            pkgs.goreleaser
          ];
        };
      };
    };
}
