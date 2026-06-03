{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
{
  packages = with pkgs; [
    go
    gitleaks
    golangci-lint
    goreleaser
    just
    git
  ];

  env.CGO_ENABLED = "0";

  scripts.build.exec = "just build";
  scripts.test.exec = "just test";
  scripts.lint.exec = "just lint";
}
