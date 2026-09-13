# repo-overview

Fast, per-module overview of a git repository — which modules exist, what their
READMEs say, and which ones actually moved in the last 24 hours.

```
repo-overview                 # the repo containing the current directory
repo-overview ~/code/jetlog   # an explicit path
repo-overview --long --since 7d --lines 10 ~/code/jetlog
```

## What it does

1. `git fetch` (skip with `--no-fetch`) — it never switches branches or touches
   your working tree, so it is safe to run with uncommitted work in progress.
2. Reports on `develop`, falling back to `main` then `master`, or `--branch X`.
3. Finds modules: directories holding a build manifest (`pom.xml`, `go.mod`,
   `package.json`, `build.gradle`, `pyproject.toml`, `Cargo.toml`, `*.csproj`,
   …), ignoring vendored trees like `node_modules` and `target`. A repo with no
   manifests falls back to its top-level directories.
4. Shows the repository's own top-level `README.md` as a banner, when the root
   is not itself a module.
5. Per module: the first 30 lines of `README.md` — rendered as markdown, with
   headings, lists, code blocks, links and box-drawn tables, in the same
   terminal style as `md-viewer` — a clickable GitHub blob URL, and lines
   added/deleted plus commit count inside the window.
6. Sorts by churn, so the busiest module is at the top.

Nested modules are charged to themselves, not to their parent.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `--lines N` | 30 | README lines to show per module |
| `--long` | off | also show the last committer and commit subject |
| `--since D` | `24h` | churn window: `90m`, `24h`, `7d` |
| `--branch B` | — | report on B instead of develop/main/master |
| `--no-fetch` | off | skip the network, use local refs |
| `--no-color` | off | plain output (also honours `NO_COLOR` and non-TTY) |

## Install

```
go install ./...
```

## Tests

```
go test ./...
```

Tests build throwaway git repositories with backdated commits, so they exercise
real git rather than mocks.
