# carryon — Design Spec

**Date:** 2026-06-01
**Status:** Approved design, pre-implementation

## Summary

`carryon` is a Go CLI that gives you one unified, location-independent picker over
**all** your Claude Code and Codex conversation histories, and resumes any of them
from wherever you currently are in your terminal.

Both tools centralize their histories under `~`, but each tool's built-in picker
only surfaces sessions whose working directory matches where you launched it.
`carryon` removes that constraint: it reads both central stores, presents every
conversation in an interactive TUI with a live preview, and on selection hands the
terminal off to the native CLI (`claude`/`codex`) — already `cd`'d into the right
project directory — by replacing itself via `exec`.

## Goals

- List every Claude Code and Codex conversation, regardless of which project it
  belongs to or where you currently are.
- Interactive, fuzzy-searchable TUI with a live preview pane to identify the right
  conversation.
- Resume by handing off to the native CLI: `carryon` `chdir`s to the original
  project directory and `exec`s the real tool, then ceases to exist (same PID, same
  terminal, no lingering parent process).
- Honor the user's normal shell aliases / default flags on launch.

## Non-Goals (v1)

- No filesystem-wide crawl (histories are centralized; we read two known stores).
- No re-implementation of the agent loop / transcript rendering as a chat (we hand
  off to the native tools).
- No Codex `archived_sessions`.
- No Windows support (the `$SHELL` + `exec` handoff is Unix-only; macOS/Linux only).
- No persistent metadata cache (parse on startup; revisit only if slow).

## Key Findings: How Histories Are Stored

### Claude Code
- Location: `~/.claude/projects/<sanitized-cwd>/<session-uuid>.jsonl`
  - Directory name is the project cwd with `/` replaced by `-`
    (e.g. `-Users-mohammadalamarneh-workspace-growth-apps-hootmail3`).
  - There may be sibling sub-directories (e.g. `<uuid>/`) holding auxiliary data —
    **ignored**; we only read top-level `*.jsonl`.
- Each file is one session. Session ID = filename UUID.
- Each JSONL line carries `cwd` and `gitBranch`; messages appear as
  `{"type":"user",...}` / assistant lines. Some lines are meta
  (`isMeta:true`, `<local-command-caveat>`, tool results).
- Resume command: `claude --resume <session-id>` (alias `-r`).

### Codex
- Location: `~/.codex/sessions/YYYY/MM/DD/rollout-<timestamp>-<uuid>.jsonl`
- First line is `session_meta` carrying `payload.id` (session ID), `payload.cwd`,
  and `payload.timestamp`.
- Real user messages appear as `response_item` entries with `role:"user"`, after
  skipping the `<environment_context>` block and `developer` messages.
- No git branch recorded.
- Resume command: `codex resume <session-id>` (UUID or thread name; UUID precedence).

## Architecture

Single Go module, `carryon`:

- `cmd/carryon` — entrypoint; runs the TUI, then performs the handoff.
- `internal/model` — `Conversation` type and the `Tool` enum (Claude, Codex).
- `internal/discovery` — finds and parses session files into `[]Conversation`.
- `internal/tui` — Bubble Tea program: list, fuzzy search, live preview pane.
- `internal/launch` — pure command-construction + the thin `exec` wrapper.

Chosen approach: **native Go TUI** with Bubble Tea + Lip Gloss. No runtime
dependencies beyond the already-installed `claude`/`codex`. (Rejected: an
fzf-driven wrapper — awkward preview pane, fiddly post-selection handoff, external
dependency; and a raw tcell/termbox TUI — more boilerplate for no benefit.)

## Data Model

```go
type Tool int // Claude, Codex

type Conversation struct {
    Tool      Tool      // which CLI owns it
    SessionID string    // UUID passed to resume
    Cwd       string    // original project dir
    Branch    string    // git branch (Claude only; "" for Codex)
    Title     string    // first real user message, trimmed; "(no message)" fallback
    Modified  time.Time // file mtime; used for sort + "age"
    Path      string    // path to the .jsonl
}
```

## Discovery & Parsing

No filesystem-wide crawl. Read the two known stores:

- Claude: glob `~/.claude/projects/*/*.jsonl`.
- Codex: glob `~/.codex/sessions/*/*/*/rollout-*.jsonl`.

Parsing is kept cheap:

- **Recency sort** uses file mtime — no parse needed.
- **Row metadata** comes from a *bounded* read per file: read lines only until we
  have what we need, then stop.
  - Claude: `cwd`/`gitBranch` from the first line that carries them; title from the
    first real `role:"user"` message, skipping meta / `<local-command-caveat>` /
    tool-result lines.
  - Codex: `cwd`/`id`/timestamp from line 1 (`session_meta`); title from the first
    `role:"user"` message, skipping `<environment_context>` and `developer` blocks.
- **Preview pane** reads the *tail* of the highlighted file lazily (recent
  user/assistant turns), only when the row is highlighted.

Robustness: corrupt/unparseable lines are skipped without crashing discovery;
unreadable files (permissions) are skipped with a quiet stderr warning.

v1 parses all rows on startup (early-stop reads keep this fast for hundreds of
files). An mtime-keyed metadata cache is deferred unless startup feels slow.

## TUI

Two panes: left = scrollable conversation list; right = live preview of the
highlighted conversation. Header shows count + active filters; footer shows
keybindings.

- **Row format:** `tool  project  age  branch  title`
  (e.g. `claude  hootmail3  2d  master  "fix the bounce handler"`). Long project
  paths are shortened to the last segment or two (with `~` for home); the full path
  shows in the preview header.
- **Sort:** most-recently-modified first (only sort in v1).
- **Search:** type to fuzzy-filter live; matches project path + title + tool name.
- **Preview pane:** recent user/assistant turns; header shows tool, full cwd,
  branch, and age. If the original `cwd` no longer exists, the header shows a
  `⚠ path missing` marker.
- **Keybindings:**
  - `↑/↓` (and `j/k`) — move
  - type / `/` — fuzzy search
  - `Enter` — resume (chdir + exec handoff)
  - `t` — cycle tool filter (all → claude → codex)
  - `.` — toggle "current directory only"
  - `q` / `Ctrl-C` — quit
  - `Esc` — clear search, or quit if search empty
- **Empty states:** friendly "no conversations found" when nothing exists; a
  distinct "no matches" when a filter empties the list.

## Launch Handoff

Order matters so the terminal is clean:

1. On `Enter`, the TUI records the selected `Conversation` and quits.
2. After Bubble Tea fully exits and **restores the terminal**, `main()` performs the
   handoff. (Never `exec` while the TUI still owns the terminal in raw mode.)
3. `main()` resolves `$SHELL`, validates the cwd, `chdir`s, then `syscall.Exec`s the
   shell with a **constant** command string and the session ID as a positional arg:
   - Claude: `exec $SHELL -ic 'exec claude --resume "$1"' carryon <id>`
   - Codex:  `exec $SHELL -ic 'exec codex resume "$1"' carryon <id>`
   - `-i` makes the shell source the user's rc, so aliases
     (`--dangerously-skip-permissions`, `--yolo`) apply. The inner `exec` means no
     shell lingers; `carryon` is replaced entirely (same PID/terminal).

### Edge Cases
- **Original cwd deleted/moved:** flagged `⚠ path missing` in the preview. On resume,
  print a warning and launch from the *current* directory instead of chdir'ing
  (resume is keyed by session ID, so it still works; only the cwd context differs).
- **Session with no user message:** title falls back to `(no message)`.
- **Shell unresolvable or `exec` fails:** print the error, exit non-zero.

## Security

The only genuinely untrusted inputs are the **session ID** and **cwd** — both can be
influenced by a malicious `.jsonl` dropped into `~/.claude`/`~/.codex` (cloned repo,
shared machine, hostile tooling). Both are kept out of the shell parser entirely.

1. **Command injection via session ID — primary risk.** The per-tool command string
   is a **compile-time constant** (`exec claude --resume "$1"`); the ID is passed as
   a separate argv element bound to `$1`. The shell parses fixed code; the ID never
   reaches the parser as code. Verified in zsh and bash: a
   `'; touch /tmp/CARRYON_PWNED; echo '` payload is treated as literal text.
   **Defense-in-depth:** validate the ID against a strict UUID regex before launch;
   refuse otherwise. Two independent layers.
2. **cwd:** applied via Go's `os.Chdir`, never in the shell string. `Stat`'d first
   (must exist and be a directory), else current-dir fallback with a warning.
3. **`$SHELL` trust:** validated as an absolute path to an existing, executable file;
   fall back to `/bin/sh` if unset or suspicious.
4. **`-i` sources rc, PATH resolves the binary:** intended — identical to what the
   user gets typing `claude` themselves; not a new risk surface.
5. **Tool name** is never derived from data — selected from the internal `Tool` enum,
   mapping to a fixed literal.

## Testing (TDD)

- **`internal/discovery`** — table-driven tests over fixture `.jsonl` files
  (real-shaped Claude and Codex lines): correct extraction of tool, session ID, cwd,
  branch, title; skipping of meta / `<local-command-caveat>` / `<environment_context>`
  / developer lines for the title; corrupt lines skipped without crashing; empty-title
  `(no message)` fallback. Plus an integration test pointing discovery at a temp dir
  laid out like `~/.claude/projects` and `~/.codex/sessions`.
- **`internal/launch`** — security-critical logic extracted into a pure function
  `buildLaunch(conv, env) → (shellPath, argv, dir, warnings, error)`, no exec:
  rejects non-UUID session IDs; confirms the ID lands as a positional arg, never in
  the command string; `$SHELL` validation + `/bin/sh` fallback; missing-cwd →
  current-dir fallback + warning. The actual `syscall.Exec` stays a one-line wrapper,
  exercised by one integration test that execs `echo` instead of the real CLI.
- **`internal/tui`** — drive Bubble Tea's `Update` directly: keystrokes narrow the
  fuzzy filter; `t` cycles the tool filter; `.` toggles current-dir-only; `Enter`
  records the selection; empty-list states. Light substring checks on the rendered
  view.
