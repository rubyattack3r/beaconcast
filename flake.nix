{
  description = "Beaconcast: animated Cobalt Strike Beacon console demos";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      pkgsFor = system: nixpkgs.legacyPackages.${system};
      version = "0.1.0";
      packagesFor =
        system:
        let
          pkgs = pkgsFor system;
          beaconcast = pkgs.buildGoModule {
            pname = "beaconcast";
            inherit version;

            src = pkgs.lib.cleanSource ./.;
            vendorHash = "sha256-oXRFTNWbbPpyveNAwY9EzUYcWTgoRcnNUJQkiPghbbU=";
            subPackages = [ "cmd/beaconcast" ];

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];

            # buildGoModule only tests subPackages by default; check everything.
            doCheck = true;
            checkPhase = ''
              runHook preCheck
              go vet ./...
              go test ./...
              runHook postCheck
            '';

            meta = {
              description = "Render scripted Cobalt Strike Beacon console demos as animated SVG and GIF";
              homepage = "https://github.com/rubyattack3r/beaconcast";
              license = pkgs.lib.licenses.mit;
              mainProgram = "beaconcast";
              platforms = pkgs.lib.platforms.unix;
            };
          };
        in
        {
          default = beaconcast;
          inherit beaconcast;
        };
    in
    {
      packages = forAllSystems packagesFor;

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/beaconcast";
          meta.description = "Render a Beacon console demo from a scene file";
        };
      });

      checks = forAllSystems (system: {
        inherit (self.packages.${system}) beaconcast;
      });

      devShells = forAllSystems (
        system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.actionlint
              pkgs.go
              pkgs.golangci-lint
              pkgs.goreleaser
              pkgs.gopls
              pkgs.gotools
              pkgs.govulncheck
              pkgs.nixfmt
            ];
          };
        }
      );

      formatter = forAllSystems (system: (pkgsFor system).nixfmt);
    };
}
