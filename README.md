# Bingo

A real-time, multiplayer Bingo web application.

## Running Locally

You can run the application directly using Go:

```bash
go run ./cmd/server/main.go
```

By default, the server will start on `http://localhost:8080`.

Alternatively, if you have Nix installed, you can drop into the development shell:

```bash
nix develop
go run ./cmd/server/main.go
```

## NixOS Deployment

This repository provides a Nix flake that outputs a NixOS module, making it trivial to host on a VPS.

1. Add the flake to your system inputs.
2. Enable the service in your NixOS configuration:

```nix
services.bingo = {
  enable = true;
  domain = "bingo.yourdomain.com"; # Automatically sets up Nginx reverse proxy with WebSocket support
  port = 8080;
};
```

3. Run `nixos-rebuild switch`.
