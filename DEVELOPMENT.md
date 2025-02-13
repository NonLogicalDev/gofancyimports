# Development Notes:

## PreRequisites

* golang
* goreleaser

## Optional: CI Toolchain

* nix (with flakes enabled)
* just (for running nix related recipes in ./nix folder)

## Nix Hermetic Builds

The repository is devShell nix enabled, this is to ensure that we can locally reproduce an exact copy of the CI environment.

To run a command in a nix hermetic environment:

```
just ./nix/hrun $COMMAND $ARGS
```

Examples:

```
just ./nix/hrun make build
just ./nix/hrun make test
```

To run a shell for debugging (bash):

```
just ./nix/hshell
```
