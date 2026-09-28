# Robert
Don't let your agent make stuff up.


## Why
Documentation is often out of date, source code is not. Robert makes it easy to manage a library of local repositories so that your agent can simply source dive for the answer instead of making it up or relying on out of date documentation.

## Install

Robert requires Git on your `PATH`. Downloaded executables do not require Go.

Download an archive and `checksums.txt` from [GitHub Releases](https://github.com/isthatcentered/robert/releases).
While the repository is private, you must sign into a GitHub account with access.

| Your machine | Archive suffix |
| --- | --- |
| Linux, Intel/AMD 64-bit | `linux_amd64.tar.gz` |
| Linux, ARM 64-bit | `linux_arm64.tar.gz` |
| macOS, Intel | `darwin_amd64.tar.gz` |
| macOS, Apple Silicon | `darwin_arm64.tar.gz` |

For example, for version 0.1.0 on Linux Intel/AMD, run these commands from the download directory:

```sh
shasum -a 256 robert_0.1.0_linux_amd64.tar.gz
# Compare the hash with this archive's entry in checksums.txt.
tar -xzf robert_0.1.0_linux_amd64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 755 robert "$HOME/.local/bin/robert"
export PATH="$HOME/.local/bin:$PATH"
robert --version
```

Use the filename for your version and machine. Add the `export PATH` line to your shell's startup file
(`~/.zshrc` or `~/.bashrc`) to keep it in future terminals.

If you previously installed Robert using Go, remove the old executable from your Go binary directory
(usually `~/go/bin/robert`) before installing the download. `type -a robert` shows which copies your
shell can find. Your catalogue in `~/.robert` and checkouts in `~/.agents/robert` are retained when you
replace or remove the executable. To upgrade, install the newer release using the same steps.

### Install the agent skill

Each archive includes `SKILL.md`. Copy it into your agent's global skills directory under
`view-source-code/SKILL.md`. For example, if your agent uses `~/.agents/skills`:

```sh
mkdir -p "$HOME/.agents/skills/view-source-code"
cp SKILL.md "$HOME/.agents/skills/view-source-code/SKILL.md"
```

Optionally add this to your global `AGENTS.md`:

> Use the `view-source-code` skill as your primary way to learn how to use a library or tool, or understand its implementation, whenever you can access and clone its repository. Fallback to context7 MCP when otherwise.

### Build from source

With the Go version specified in `go.mod` installed, run this from a checkout:

```sh
go install ./cmd/robert
```

Ensure your Go binary directory is on `PATH`. Source builds report `robert dev`; release builds
report the version embedded by GoReleaser.

## Quickstart

```sh
robert add spf13/cobra
robert add goreleaser/goreleaser --tag v2.18.2
robert list
robert update
robert remove spf13/cobra
```

`robert update` updates the repositories in your library. To upgrade Robert itself, install a newer release.
Run `robert --help` or `robert <command> --help` for more options.

## Development and releases

See [the release guide](docs/releasing.md) for checks, local release rehearsals, and publishing a version.

## License

[MIT](LICENSE)
