<p align="center">
  <img src="docs/banner.svg" alt="git-cleaner" width="520">
</p>

<p align="center">
  <b>Prune stale Git branches across every repository on your machine, in one command.</b>
</p>

<p align="center">
  <a href="https://github.com/CptConi/git-cleaner/actions/workflows/ci.yml"><img src="https://github.com/CptConi/git-cleaner/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/CptConi/git-cleaner/releases/latest"><img src="https://img.shields.io/github/v/release/CptConi/git-cleaner?sort=semver" alt="Latest release"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/CptConi/git-cleaner" alt="Go version"></a>
  <img src="https://img.shields.io/badge/platforms-macOS%20%7C%20Linux%20%7C%20Windows-informational" alt="Platforms: macOS, Linux, Windows">
  <a href="LICENSE"><img src="https://img.shields.io/github/license/CptConi/git-cleaner" alt="MIT license"></a>
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#how-it-works">How it works</a> ·
  <a href="#safety">Safety</a> ·
  <a href="#about-disk-space">Disk space</a>
</p>

---

Months of work leave every repository cluttered with local branches: merged
features, abandoned experiments, branches deleted on the remote long ago.
**git-cleaner** walks a folder such as `~/Projects`, finds every Git
repository in it and, in each one:

1. prunes the remote-tracking branches whose branch was deleted on the remote
   (`git fetch --all --prune`);
2. force-deletes (`git branch -D`) every local branch that is not on your keep
   list (`main`, `master`, `dev` and `develop` by default) and whose commits
   are all on a remote: unpushed work is never deleted;
3. reports what it did and how much disk space Git can reclaim, and can run
   `git gc` to reclaim it right away.

Nothing is ever pushed: branches on your remotes are never touched.

<p align="center">
  <img src="docs/demo.svg" alt="Output of git-cleaner --dry-run on four repositories" width="760">
</p>

## Features

- 🔍 **Finds every repository** under a folder, skipping `node_modules`,
  worktrees and submodules.
- ✂️ **Prunes stale remote-tracking branches**, then deletes local branches,
  merged or not, once their commits are safe on a remote.
- 🛡️ **Never deletes unpushed work**: a branch holding commits that no remote
  has is kept. So are the branches of your keep list (glob patterns such as
  `release/*`) and the checked-out ones, and `--exclude` leaves whole folders
  alone.
- 👀 **`--dry-run`** shows everything that would happen, without changing
  anything.
- 💾 **Disk space report**: an estimate of what Git can reclaim, and the real
  numbers with `--gc`.
- ⚡ **Fast and portable**: repositories are processed in parallel. It is a
  single static binary with no dependency but Git, for macOS, Linux and
  Windows.

## Install

**With Go** (1.22 or later):

```sh
go install github.com/CptConi/git-cleaner@latest
```

The binary lands in `$(go env GOPATH)/bin` (`~/go/bin`, or
`%USERPROFILE%\go\bin` on Windows): make sure that folder is on your `PATH`.

**Prebuilt binaries**: download the archive for your platform from the
[latest release](https://github.com/CptConi/git-cleaner/releases/latest),
extract it and move `git-cleaner` (`git-cleaner.exe` on Windows) to a folder
on your `PATH`. On macOS, a binary downloaded with a browser is quarantined by
Gatekeeper: run `xattr -d com.apple.quarantine git-cleaner` once.

**From source**:

```sh
git clone https://github.com/CptConi/git-cleaner.git
cd git-cleaner
make install        # Windows: go install .
```

Git 2.23 or later must be on the `PATH` (2.31 or later for the disk space
estimate). If you need to set up Git and Go first:

<details>
<summary><b>macOS</b></summary>

```sh
brew install git go
echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.zshrc   # for go install
```

Apple's Git (`xcode-select --install`) works too.
</details>

<details>
<summary><b>Linux</b></summary>

```sh
sudo apt install git make            # Fedora: sudo dnf install git make
# Distribution packages of Go are often outdated: use the official archive
curl -LO https://go.dev/dl/go1.27.1.linux-amd64.tar.gz   # latest: https://go.dev/dl/
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.27.1.linux-amd64.tar.gz
echo 'export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"' >> ~/.bashrc
```
</details>

<details>
<summary><b>Windows</b> (PowerShell)</summary>

```powershell
winget install --id Git.Git -e
winget install --id GoLang.Go -e
# open a new terminal so that PATH is refreshed
```

Colors and the banner show up in Windows Terminal, PowerShell and cmd
(Windows 10 or later). Git Bash (mintty) gets plain text.
</details>

## Usage

```text
git-cleaner [options] <root-directory>
```

Always start with a dry run, then apply:

```sh
git-cleaner --dry-run ~/Projects
git-cleaner ~/Projects
```

| Option | Default | Description |
|---|---|---|
| `--keep <list>` | `main,master,dev,develop` | Local branches never deleted: comma-separated names or glob patterns. Replaces the default list; can be repeated. `*` does not match `/`: `release/*` protects `release/1.0`, not `release/1.0/fix`. |
| `--exclude <list>` | none | Folders skipped with everything below them: comma-separated glob patterns relative to the root, with `/` as the separator on every OS, case-sensitive. `acme` covers `acme/web/app`; `portfolio` does not match `personal/portfolio` (patterns start at the root). Can be repeated. |
| `--dry-run` | off | Show what would be pruned and deleted; change nothing. |
| `--no-fetch` | off | Skip the fetch/prune step (no network access): branches are checked against the remotes as of the last fetch. |
| `--gc` | off | Run `git gc` in each repository after the cleanup and report the space actually freed ([details](#about-disk-space)). |
| `-j`, `--jobs <n>` | CPUs, max 8 | Repositories processed in parallel. |
| `--timeout <d>` | `2m` | Time limit of each network operation (`30s`, `5m`...); `0` disables it. |
| `--no-color` | off | Plain output. `NO_COLOR` is honored too, and `FORCE_COLOR=1` keeps colors in a pipe (`\| less -R`). |
| `--version`, `-h` | | Print the version / the help. |

Flags go before or after the directory:

```sh
git-cleaner --keep 'main,develop,release/*' ~/Projects      # quote patterns for the shell
git-cleaner --exclude 'acme,personal/portfolio' ~/Projects  # leave these folders alone
git-cleaner --no-fetch --gc ~/Projects                      # offline, then reclaim space
git cleaner --dry-run ~/Projects                            # Git runs git-cleaner as a subcommand
```

| Exit status | Meaning |
|---|---|
| `0` | Success (branches skipped because they are checked out included). |
| `1` | The run completed, but some operations failed (fetch, deletion, gc). |
| `2` | Invalid arguments or unusable environment: nothing was done. |

## How it works

1. **Discovery**: the root folder is walked recursively. A folder containing a
   `.git` *directory* is a repository, and the scan does not descend into it
   (nested clones are usually vendored dependencies or tool caches). A `.git`
   *file* (linked worktree, submodule) is skipped too, because its branches
   belong to the main repository. `node_modules` folders are ignored, symbolic
   links are not followed, and unreadable folders are reported and skipped.
   Folders matching `--exclude` are skipped with their whole subtree, which is
   not even read; they are listed after the scan, and a pattern that matches
   no folder is reported before anything is processed.
2. **Prune**: `git fetch --all --prune` removes the local remote-tracking
   references (`origin/feature-x`) whose branch no longer exists on the
   server. In dry-run mode, `git remote prune --dry-run <remote>` lists them
   instead: it only asks the remote for its branch list and writes nothing.
   A network or authentication failure is reported, and the cleanup of that
   repository goes on.
3. **Sorting**: each local branch is either *kept* because it matches
   `--keep`, *skipped* because it is checked out (in the working tree or a
   linked worktree, which Git refuses to delete), *kept* because it holds
   **commits that no remote has**, or *deleted*. A commit counts as pushed when
   a remote-tracking branch that survives the prune contains it: branches
   merged on the server and then deleted qualify, while work never pushed,
   commits added after the last push and branches deleted on the remote
   without being merged are kept.
4. **Deletion**: `git branch -D` runs branch by branch. When Git refuses
   (locked reference, branch being rebased...), a warning is printed and the
   next branch is processed.
5. **Report**: the summary gives the counts and estimates the disk space held
   only by the removed references (`git rev-list --disk-usage`).
6. **Optional `git gc`** (`--gc`): one repository at a time, measuring the
   disk usage of `.git` before and after (allocated blocks on macOS and Linux,
   like `du`; file sizes on Windows).

Repositories are processed in parallel, and results are printed in scan order.

## Safety

- Nothing is ever pushed: branches on the remotes are never modified.
- A local branch is only deleted when every one of its commits is on a
  remote, so a deleted branch can always be fetched back. The real run checks
  this right after fetching; `--dry-run` and `--no-fetch` rely on the last
  fetch (a branch merged on the server since then shows up as kept).
- `--dry-run` changes nothing: it only contacts the remotes to list their
  branches. Run it first to review what will go.
- The checked-out branch and branches used by linked worktrees are never
  deleted. Bare repositories, submodules and worktrees are not processed.
- Git runs non-interactively: `GIT_TERMINAL_PROMPT=0` makes it fail instead of
  waiting for a password, `GCM_INTERACTIVE=never` keeps Git Credential Manager
  from opening sign-in windows (unless you set that variable yourself), and
  `--timeout` bounds network operations (on macOS and Linux, git is first
  asked to stop gracefully so that it removes its lock files). Use an SSH
  agent or stored credentials for private remotes.
- Environment variables such as `GIT_DIR` or `GIT_WORK_TREE`, which would
  send Git to another repository, are removed from the environment of every
  git command.
- Git output is parsed with `LC_ALL=C`, so the language of the system does not
  matter.

## Recovering a deleted branch

Deleted branches only hold pushed commits, so nothing is lost. If the branch
still exists on the remote, `git switch <branch>` recreates it. Otherwise its
commits live on in another remote branch (typically the one it was merged
into), and the commit printed in the report brings it back:

```sh
git -C ~/Projects/api branch feature/login 71db8c968a
```

## About disk space

A branch is a 41-byte file: deleting it frees nothing by itself. The space is
held by the objects (commits, trees, files) it references, which Git's garbage
collection deletes only when all three conditions hold:

1. **No reference points to them any more.** The estimate therefore leaves out
   objects still reachable from a remote branch, a tag, another branch or
   HEAD. As git-cleaner only deletes pushed branches, the space mostly comes
   from pruned remote-tracking branches whose commits nothing else references.
2. **No reflog entry references them.** By default, entries expire after 30
   days for commits no longer reachable from their branch
   (`gc.reflogExpireUnreachable`), and after 90 days otherwise.
3. **They are older than two weeks** (`gc.pruneExpire`).

Git runs this collection automatically from time to time (`git gc --auto`).
`--gc` runs `git gc` now: it repacks loose objects and deletes the expired
ones, which frees a lot of space in repositories that were never collected. It
keeps the objects that reflogs still reference. On small repositories, `.git`
may even grow slightly (commit-graph, pack indexes).

To reclaim everything immediately in one repository, at the cost of the
recovery safety net (the reflog history of every branch is lost too):

```sh
git -C <repository> reflog expire --expire-unreachable=now --all
git -C <repository> gc --prune=now
```

## Development

```sh
go test ./...      # unit and integration tests (they drive the real git)
make vet           # go vet for every target platform
make build         # ./git-cleaner for this machine
make release       # cross-compile every platform into dist/
```

Continuous integration runs the tests on Linux, macOS and Windows, with the
oldest supported Go version and the latest one. Pushing a `v*` tag (e.g.
`v1.2.0`) makes the [release workflow](.github/workflows/release.yml) build
the binaries with [GoReleaser](https://goreleaser.com) and publish them on the
release page.

| File | Content |
|---|---|
| `main.go` | Command line, orchestration, parallel workers |
| `scan.go` | Repository discovery |
| `cleaner.go` | Per-repository cleanup: prune, sort, delete, estimate, gc |
| `git.go` | Git execution (environment, timeouts, errors) and commands |
| `whitelist.go` | `--keep` and `--exclude` pattern parsing and matching |
| `report.go`, `banner.go` | Console output, summary and startup banner |
| `platform_*.go` | OS specifics: terminal colors, disk usage of files |

### Design notes

- **Why Go**: a single static executable per OS and architecture, with nothing
  to install on the target machine (unlike Node.js or Python). Cross-compiling
  every platform from one machine is built in, the standard library covers
  everything (no third-party dependency), startup is instantaneous and
  goroutines make parallel fetches easy. Rust would offer the same deployment
  model at the cost of a steeper learning curve and slower builds, for no gain
  in a tool that mostly waits for git.
- **Why drive the git executable** rather than a Git library such as go-git:
  your own setup (credential helpers, SSH agent and configuration, proxies,
  `safe.directory`, hooks) works exactly as on the command line, with the
  exact behavior of `git fetch --prune` and `git branch -D`.

## Contributing

- `main` holds the released code and only receives merges from `dev`, through
  a pull request.
- `dev` is the integration branch. Each feature lives on its own branch
  (`feature/<name>`), merged into `dev` once validated.
- Features are spec-driven with [OpenSpec](https://github.com/Fission-AI/OpenSpec):
  each one starts as a change proposal in [`openspec/changes`](openspec/changes)
  (`/opsx:propose`), is implemented on its feature branch (`/opsx:apply`) and
  is archived into [`openspec/specs`](openspec/specs) once merged
  (`/opsx:archive`).
- Commit messages fit on one line: `<type> (<feature>) <what changed>`, e.g.
  `feat (keep-patterns) add glob support, wording for help`.
- The `tests passed` check (the whole CI matrix) is required on `dev` and
  `main`, and pull requests into `main` must come from `dev` (the `source is
  dev` check). The branch rulesets are versioned in
  [`.github/rulesets`](.github/rulesets): *Settings → Rules → Rulesets → New
  ruleset → Import a ruleset*.

## License

[MIT](LICENSE) © 2026 Nicolas Renard
