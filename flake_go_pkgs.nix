{pkgs ? <nixpkgs>, version, vendorHash, srcDir, ...}:
{
  gofancyimports = pkgs.buildGoModule {
    pname = "gofancyimports";
    version = version;
    src = srcDir;
    vendorHash = vendorHash;
    subPackages = [ "cmd/gofancyimports" ];
    ldflags = [ "-s" "-w" ];
    meta = with pkgs.lib; {
      description = "No-Compromise Deterministic GoLang Import Management";
      homepage = "https://github.com/NonLogicalDev/gofancyimports";
      license = licenses.mit;
      maintainers = [];
    };
  };
}