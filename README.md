

<p align="center"><img src="./logo.png" alt="tada" width="120" /></p>

![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)
![Terminal](https://img.shields.io/badge/Interface-CLI-informational)
![License](https://img.shields.io/badge/License-MIT-green)

A fast, minimalist **to‑do manager for the terminal**.  
It ships as a single Go binary and supports color themes and grouped output.

---

## Features

- **Simple commands** for daily use (see `todo help` for the full list).
- **Color output** with themes: `classic`, `neon`, `mono`.
- **Force/disable colors** regardless of TTY detection.
- **Group** items by pending/done in the list view.
- Clean exit codes (non‑zero on error).

> Commands are routed by an internal CLI runner; run `todo help` to see what’s available in your build (e.g., `add`, `list`, `done`, `rm`, `edit`, …).

---

## Install

From source (recommended during development):

```bash
# If the main package is at the repo root:
go build -o todo .

# If the main package lives under cmd/todo:
go build -o todo ./cmd/todo
```

Optionally, with Go modules and a public repo path:

```bash
go install github.com/idilsaglam/todo/cmd/todo@latest
```

---

## Usage

```text
todo [global flags] <command> [args...]
```

**Global flags (apply to every subcommand):**

```text
-color            force color output even when not a TTY
-no-color         disable color output
-theme string     ui theme: classic|neon|mono (default "classic")
-group            group output by pending/done
```

Show help:

```bash
todo help
```

---

## Examples (subject to available commands)

> The exact subcommands depend on `internal/cli`. These are common patterns.

Add a task:

```bash
todo add "Buy coffee beans ☕"
```

List tasks:

```bash
todo list
# or grouped view:
todo -group list
```

Mark as done (by ID):

```bash
todo done 3
```

Remove a task:

```bash
todo rm 3
```

Edit a task:

```bash
todo edit 3 "Buy single‑origin beans"
```

Switch theme:

```bash
todo -theme neon list
todo -no-color list   # disable colors
todo -color list      # force colors even when piped
```

---

## Project layout

```
.
├─ internal/
│  ├─ cli/     # command routing, argument parsing, help text
│  └─ ui/      # theming, color forcing, formatting
└─ main.go     # root flags → ui config → cli.Run
```

---

## Exit codes

- `0` — success  
- non‑zero — an error occurred (the CLI prints the message; `main` exits with that code)

