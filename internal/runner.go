package internal

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Options tune output behavior from root flags.
type Options struct {
	Group bool // list grouped by pending/done (for a future non-TUI list view)
}

// ---------------------------------------------------
// CLI router
// ---------------------------------------------------

func Run(args []string, opt Options) int {
	if len(args) == 0 {
		PrintHelp()
		return 2
	}
	cmd, a := args[0], args[1:]

	switch cmd {
	case "help", "-h", "--help":
		PrintHelp()
		return 0

	case "ls":
		return doList(opt)

	case "add":
		if len(a) == 0 {
			fail("usage: todo add <title...>")
			return 2
		}
		return doAdd(strings.Join(a, " "))

	case "done":
		if len(a) != 1 {
			fail("usage: todo done <index>")
			return 2
		}
		n, err := strconv.Atoi(a[0])
		if err != nil {
			fail("done: not a number: " + a[0])
			return 2
		}
		return doToggle(n)

	case "rm":
		if len(a) != 1 {
			fail("usage: todo rm <index>")
			return 2
		}
		n, err := strconv.Atoi(a[0])
		if err != nil {
			fail("rm: not a number: " + a[0])
			return 2
		}
		return doRemove(n)

	case "auth":
		if len(a) == 0 {
			fail("usage: todo auth <login|logout|status|whoami>")
			return 2
		}
		switch a[0] {
		case "login":
			return doAuthLogin()
		case "logout":
			return doAuthLogout()
		case "status":
			return doAuthStatus()
		case "whoami":
			return doAuthWhoAmI()
		default:
			fail("usage: todo auth <login|logout|status|whoami>")
			return 2
		}
	}

	fail("unknown subcommand: " + cmd)
	fmt.Fprintln(os.Stderr)
	PrintHelp()
	return 2
}

func PrintHelp() {
	fmt.Printf(`todo - a tiny CLI

Usage:
  todo <subcommand> [args]

Subcommands:
  add <title...>     Add a new item (title can be multiple words)
  ls                 List items (interactive TUI)
  done <index>       Toggle done for item at 1-based index
  rm <index>         Remove item at 1-based index
  auth <login|logout|status|whoami>   Token authentication

Examples:
  todo add "Buy milk"
  todo ls
  todo done 2
  todo rm 3
`)
}

// ---------------------------------------------------
// Auth subcommands (use functions from auth.go)
// ---------------------------------------------------

func doAuthLogin() int {
	fmt.Print("Paste your token: ")
	var token string
	if _, err := fmt.Scanln(&token); err != nil {
		fail("read token: " + err.Error())
		return 1
	}
	if err := SetToken(token, nil); err != nil {
		fail("save token: " + err.Error())
		return 1
	}
	ok("logged in")
	return 0
}

func doAuthLogout() int {
	ti, _ := GetToken()
	if ti != nil && ti.Source == "env" {
		ok("token is provided by TADA_TOKEN env var (nothing to delete)")
		return 0
	}
	if err := DeleteToken(); err != nil {
		fail("logout: " + err.Error())
		return 1
	}
	ok("logged out")
	return 0
}

func doAuthStatus() int {
	ti, _ := GetToken()
	if ti == nil {
		fmt.Println(mutedStyle.Render("not logged in"))
		fmt.Println("Run: todo auth login")
		return 0
	}
	fmt.Printf("source: %s\n", ti.Source)
	if ti.ExpiresAt != nil {
		fmt.Printf("expires: %s\n", ti.ExpiresAt.UTC().Format(time.RFC3339))
	} else {
		fmt.Println("expires: (unknown)")
	}
	fmt.Println("env override: TADA_TOKEN")
	return 0
}

// whoami tries to decode JWT locally (unsigned); opaque tokens print basic info.
func doAuthWhoAmI() int {
	ti, _ := GetToken()
	if ti == nil {
		fail("not logged in. Run: todo auth login")
		return 2
	}
	token := ti.Token
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		payloadB64 := parts[1]
		// add padding if needed
		switch len(payloadB64) % 4 {
		case 2:
			payloadB64 += "=="
		case 3:
			payloadB64 += "="
		}
		if p, err := decodeB64URL(payloadB64); err == nil {
			fmt.Println("JWT payload:")
			fmt.Println(p)
			return 0
		}
	}
	fmt.Println("Opaque token (cannot introspect locally).")
	fmt.Println("source:", ti.Source)
	return 0
}

func decodeB64URL(s string) (string, error) {
	dec, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		dec2, err2 := base64.URLEncoding.DecodeString(s)
		if err2 != nil {
			return "", err
		}
		return string(dec2), nil
	}
	return string(dec), nil
}

// Require a token for future networked commands.
func ensureAuth() (*TokenInfo, int) {
	ti, _ := GetToken()
	if ti == nil || strings.TrimSpace(ti.Token) == "" {
		fail("no token found. Set TADA_TOKEN or run `todo auth login`")
		return nil, 2
	}
	return ti, 0
}

// ---------------------------------------------------
// Core subcommands (local JSON CRUD)
// ---------------------------------------------------

func doList(opt Options) int {
	items, err := Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	// The interactive TUI (now defined in tui.go). It will save on quit if changed.
	if err := runInteractiveList(items, opt); err != nil {
		fail("tui: " + err.Error())
		return 1
	}
	return 0
}

func doAdd(title string) int {
	items, err := Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	title = strings.TrimSpace(title)
	if title == "" {
		fail("add: empty title")
		return 2
	}
	items = append(items, Item{Title: title})
	if err := Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("added")
	return 0
}

func doToggle(userIndex int) int {
	items, err := Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	if userIndex < 1 || userIndex > len(items) {
		fail(fmt.Sprintf("index out of range: have %d, got %d", len(items), userIndex))
		fmt.Fprintln(os.Stderr, mutedStyle.Render("Hint: run `todo ls` to see valid indexes"))
		return 2
	}
	idx := userIndex - 1
	items[idx].Done = !items[idx].Done
	if err := Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("toggled")
	return 0
}

func doRemove(userIndex int) int {
	items, err := Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	if userIndex < 1 || userIndex > len(items) {
		fail(fmt.Sprintf("index out of range: have %d, got %d", len(items), userIndex))
		fmt.Fprintln(os.Stderr, mutedStyle.Render("Hint: run `todo ls` to see valid indexes"))
		return 2
	}
	idx := userIndex - 1
	items = append(items[:idx], items[idx+1:]...)
	if err := Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("removed")
	return 0
}
