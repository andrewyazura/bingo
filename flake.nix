{
  description = "bingo";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "bingo";
            version = "0.1.0";
            src = ./.;
            vendorHash = "sha256-gd1OBd1grvaNV2zCedPvtQ0+yRVXyN6+ijcpaTNVw+E=";
            subPackages = [ "cmd/server" ];
            postInstall = ''
              mkdir -p $out/share/bingo
              cp schema.sql $out/share/bingo/
              cp -r web $out/share/bingo/
            '';
          };
        }
      );
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gopls
              golangci-lint
              postgresql
              podman
              podman-compose
            ];

            shellHook = ''
              echo "Go version: $(go version)"
              exec zsh
            '';
          };
        }
      );

      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        with lib;
        let
          cfg = config.services.bingo;
        in
        {
          options.services.bingo = {
            enable = mkEnableOption "Bingo service";
            package = mkOption {
              type = types.package;
              default = self.packages.${pkgs.system}.default;
              description = "The bingo package to use";
            };
            port = mkOption {
              type = types.port;
              default = 8080;
              description = "Port to listen on";
            };
            databaseUrl = mkOption {
              type = types.str;
              default = "postgres://postgres:postgres@localhost:5432/bingo?sslmode=disable";
              description = "Database URL";
            };
            domain = mkOption {
              type = types.str;
              default = "bingo.local";
              description = "Domain name for the Nginx virtual host";
            };
          };

          config = mkIf cfg.enable {
            systemd.services.bingo = {
              description = "Bingo backend service";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
              environment = {
                PORT = toString cfg.port;
                DATABASE_URL = cfg.databaseUrl;
              };
              serviceConfig = {
                ExecStart = "${cfg.package}/bin/server";
                WorkingDirectory = "${cfg.package}/share/bingo";
                Restart = "on-failure";
                DynamicUser = true;
              };
            };

            services.nginx = {
              enable = true;
              upstreams."bingo_backend" = {
                servers = {
                  "127.0.0.1:${toString cfg.port}" = { };
                };
                extraConfig = ''
                  hash $uri consistent;
                '';
              };
              virtualHosts."${cfg.domain}" = {
                locations."/" = {
                  proxyPass = "http://bingo_backend";
                  proxyWebsockets = true;
                };
              };
            };
          };
        };

    };
}
