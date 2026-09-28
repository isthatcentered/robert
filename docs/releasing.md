# Releasing Robert

A version tag identifies the exact commit being released. Pushing a `v*` tag starts the Release
workflow, which runs all checks and then uses GoReleaser to publish archives and checksums to
GitHub Releases. Normal branch pushes and pull requests run checks without publishing.

```text
Push version tag -> Linux/macOS checks -> Build four executables -> Publish GitHub Release
```

## Configuration

- `.goreleaser.yaml` builds `./cmd/robert` for Linux and macOS, each on amd64 and arm64.
  Archives contain the executable, README, MIT license, and agent skill. The release version is
  embedded in `robert --version`.
- `.github/workflows/ci.yml` runs formatting, vet, race tests, and CLI QA on Linux and macOS.
  A separate job validates GoReleaser configuration and rehearses all four release builds.
- `.github/workflows/release.yml` reuses those checks before publishing. It fetches full Git history,
  reads the Go version from `go.mod`, and runs GoReleaser 2.18.2.

GitHub Actions must be enabled for the repository. The release job uses GitHub's automatic
`GITHUB_TOKEN` with `contents: write`; no personal access token or GoReleaser Pro license is needed.
Private repository releases are accessible only to people with repository access. Publishing a
release does not change repository visibility.

## Check and rehearse locally

Install [GoReleaser OSS](https://goreleaser.com/getting-started/install/oss/), version 2.18.2 to match CI,
and the Go version in `go.mod`. Git and Python 3 are also needed for the checks.

```sh
gofmt -l cmd internal  # Must print no files.
go vet ./...
go test -race ./...
python3 scripts/qa_cli.py --allow-finding silent-redundant-clone-cleanup-failure
goreleaser check
goreleaser release --snapshot --clean
```

The snapshot command creates archives in `dist/` without publishing or requiring a version tag.
`--clean` replaces the previous contents of `dist/`. Extract the archive for your machine into a
temporary directory and run its `robert --version` and `robert --help` before publishing.

The existing QA probe can report `silent-redundant-clone-cleanup-failure`: simultaneous duplicate
adds can leave an unused checkout when filesystem permissions prevent deleting it. CI explicitly
allows that one finding while preserving it in the JSON report and logs. Any failed QA case or
other finding blocks release. Run the script without `--allow-finding` for strict reporting; it
returns 2 when it detects the known limitation. Remove this exception when cleanup is fixed.

## Publish

Commit the changes and push the branch. Check that the Checks workflow succeeds in GitHub Actions.
From the clean, tested commit, create and push the version tag:

```sh
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

Use a new version for each release. For example, use `v0.1.1` for the next patch or `v0.2.0` for
the next feature release while the CLI is evolving. Tags such as `v0.2.0-rc.1` automatically create
a GitHub prerelease.

Watch the Release workflow in GitHub Actions. Once it succeeds, check the release notes and all
four archives at https://github.com/isthatcentered/robert/releases. Download the archive for your
machine, verify its checksum, install it, and confirm `robert --version` matches the release.

If checks fail, no release is published. If publishing fails partway through, inspect the run and
any draft release before retrying. Do not move a tag that already has a published release; fix the
problem in a new commit and release a new version.

## Update the release tools

Keep the GoReleaser version in both workflows and this guide in sync. GitHub Actions are pinned
to commit hashes; update them deliberately and run the checks and snapshot rehearsal again.
