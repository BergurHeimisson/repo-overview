# repo-overview

Fast, per-module overview of a git repository — which modules exist, what their
READMEs say, and which ones actually moved in the last 24 hours.

```
repo-overview                 # the repo containing the current directory
repo-overview ~/code/jetlog   # an explicit path
repo-overview --short --since 7d --lines 10 ~/code/jetlog
repo-overview --skip-stale     # only modules that moved inside the window
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
5. Per module: the last commit (author and subject, hide with `--short`),
   the first line of `README.md` (more with `--lines N`) — rendered as markdown, with
   headings, lists, code blocks, links and box-drawn tables, in the same
   terminal style as `md-viewer` — a clickable GitHub blob URL, and lines
   added/deleted plus commit count inside the window.
6. Sorts by churn, so the busiest module is at the top. `--skip-stale` drops the
   modules with no commit inside the window entirely, and says how many it hid.
7. Pauses a screenful at a time when stdout is a terminal, like `more`: space
   for the next page, enter for one more line, `q` to stop. Piped or
   redirected output is never paged; `--no-pager` turns it off outright.

Nested modules are charged to themselves, not to their parent.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `--lines N` | 1 | README lines to show per module |
| `--short` | off | omit the last committer and commit subject |
| `--since D` | `24h` | churn window: `90m`, `24h`, `7d` |
| `--skip-stale` | off | hide modules with no commit inside the window |
| `--branch B` | — | report on B instead of develop/main/master |
| `--no-fetch` | off | skip the network, use local refs |
| `--no-color` | off | plain output (also honours `NO_COLOR` and non-TTY) |
| `--no-pager` | off | print everything at once instead of pausing per screen |

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
