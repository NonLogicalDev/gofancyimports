# ref: https://github.com/hercules-ci/flake-parts
# ref: https://ryantm.github.io/nixpkgs/builders/special/mkshell/
# ref: https://github.com/kamadorueda/alejandra
{
  description = "No-Compromise Deterministic GoLang Import Management";

  inputs = {
    nixpkgs.url = "nixpkgs";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {
    nixpkgs,
    flake-parts,
    ...
  }: let
    pkg_meta = builtins.fromJSON (builtins.readFile ./pkg_meta.json);
  in
    flake-parts.lib.mkFlake {inherit inputs;} {
      # Re-use nix-package system set:
      systems = nixpkgs.lib.systems.flakeExposed;

      # Define flake outputs:
      perSystem = {
        config,
        system,
        pkgs,
        ...
      }: {
        # Expose gofancyimports as a package
        packages = (import ./flake_go_pkgs.nix) {
          pkgs = pkgs;
          version = pkg_meta.version;
          vendorHash = pkg_meta.nixVendorHash.gofancyimports;
          srcDir = ./.;
        } // {
          default = config.packages.gofancyimports;
        };

        # Development shell
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


