{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, flake-utils, nixpkgs, ... }@inputs:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs ({
          inherit system;
        });
      in
      {
        devShell = pkgs.pkgs.mkShell {

          buildInputs = with pkgs;
            [
              go_1_22
            ];
          shellHook = ''
            export CFLAGS="-I${pkgs.glibc.dev}/include"
            export LDFLAGS="-L${pkgs.glibc}/lib"
            [ -n "$(go env GOBIN)" ] && export PATH="$(go env GOBIN):''${PATH}"
            [ -n "$(go env GOPATH)" ] && export PATH="$(go env GOPATH)/bin:''${PATH}"
          '';
        };

      });
}
