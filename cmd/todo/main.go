package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/idilsaglam/todo/internal/cli"
	"github.com/idilsaglam/todo/internal/ui"
)

func main() {
	// Root flags (apply to every subcommand)
	colorAlways := flag.Bool("color", false, "force color output even when not a TTY")
	noColor := flag.Bool("no-color", false, "disable color output")
	themeName := flag.String("theme", "classic", "ui theme: classic|neon|mono")
	groupPending := flag.Bool("group", false, "group output by pending/done")

	flag.Parse()

	// Configure UI once from root flags.
	ui.SetColorForcing(*colorAlways, *noColor)
	ui.SetTheme(*themeName)

	// Hand the remaining args to the CLI runner.
	args := flag.Args()
	if len(args) == 0 {
		cli.PrintHelp()
		os.Exit(2)
	}

	code := cli.Run(args, cli.Options{
		Group: *groupPending,
	})
	if code != 0 {
		// runner already printed an error; just exit with code.
		fmt.Fprintln(os.Stderr)
	}
	os.Exit(code)
}
