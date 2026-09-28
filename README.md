# Robert
Don't let your agent make stuff up.


## Why
Documentation gets out of date, source code does not. Robert makes it easy to manage a library of local repositories so that your agent can simply source dive for the answer instead of making it up or relying on out of date documentation.

## Install

Install [Go](https://go.dev/dl/) (the version in `go.mod` or newer) and Git, then run:

```sh
go install github.com/isthatcentered/robert/cmd/robert@latest
```

Without Go, download a binary for your platform from [GitHub Releases](https://github.com/isthatcentered/robert/releases).

## Agent skill

Save [SKILL.md](SKILL.md) as `view-source-code/SKILL.md` in your agent's global skills directory
(for example, `~/.agents/skills/view-source-code/SKILL.md`).

Optionally add this to your global `AGENTS.md`:

> Use the `view-source-code` skill as your primary way to learn how to use a library or tool, or understand its implementation, whenever you can access and clone its repository. Fallback to context7 MCP when otherwise.

## Quickstart

```sh
robert add isthatcentered/robert
robert add goreleaser/goreleaser --tag v2.18.2
robert list
robert update
robert remove isthatcentered/robert
```

`robert update` updates the repositories in your library. To upgrade Robert itself, rerun the install command.
Run `robert --help` or `robert <command> --help` for more options.

## Development and releases

See [the release guide](docs/releasing.md) for checks, local release rehearsals, and publishing a version.

## License

[MIT](LICENSE)
