package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Makepad-fr/tada/internal"
)

func main() {
	// Root flags (apply to every subcommand)
	groupPending := flag.Bool("group", false, "group output by pending/done")
	flag.Parse()

	// Hand the remaining args to the CLI runner.
	args := flag.Args()
	if len(args) == 0 {
		internal.PrintHelp()
		os.Exit(2)
	}

	code := internal.Run(args, internal.Options{
		Group: *groupPending,
	})
	if code != 0 {
		fmt.Fprintln(os.Stderr)
	}
	os.Exit(code)
}
