# github_inbox_tui

A minimal GitHub inbox TUI built with Bubble Tea.

## Motivation

GitHub notifications get noisy fast. This TUI keeps your review and issue
inbox focused, fast to scan, and keyboard-driven without living in the browser.

## Requirements

- Go 1.24+
- A GitHub personal access token in `GITHUB_TOKEN`

## Run

```bash
export GITHUB_TOKEN=your_token

go run ./cmd/github_inbox_tui
```

If `GITHUB_TOKEN` is not set, the app will prompt you to enter it.
On macOS, the token is saved in Keychain. On other platforms, it is stored in
`~/.config/github_inbox_tui/token` with `0600` permissions.

## Makefile

```bash
make run
make build
make fmt
make tidy
```

## Keybindings

- ↑/↓ or j/k: navigate
- enter: details view
- o: open selected item in browser
- r: refresh
- f: cycle filters
- tab: switch PRs / Issues
- c: comment (multiline, ctrl+g to send)
- x: close/reopen (with confirmation)
- n/p: next/prev comments (detail view)
- q: quit

## Structure

- `cmd/github_inbox_tui/main.go`: entry point and program bootstrap
- `internal/app/model.go`: Bubble Tea model, update loop, and views
- `internal/app/types.go`: shared domain types and filters
- `internal/app/github.go`: GitHub API calls and helpers
- `internal/app/commands.go`: async commands (fetch, comment, close/reopen, open URL)
- `internal/app/ui.go`: Catppuccin Latte styles and UI helpers

## Filters

- Open: `is:open archived:false involves:@me`
- Review requested: `review-requested:@me`
- Assigned: `assignee:@me`
- Mentions: `mentions:@me`
- Authored: `author:@me`

## Detail View

- PRs show draft/mergeable status, review summary, and change stats.
- Issues show labels and assignees.

## License

MIT. See `LICENSE`.
