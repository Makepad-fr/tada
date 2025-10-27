package ui

import (
	"fmt"
	"os"
)

var (
	reset = "\033[0m"
	bold  = "\033[1m"
	dim   = "\033[2m"

	fgGray   = "\033[90m"
	fgGreen  = "\033[32m"
	fgYellow = "\033[33m"
	fgBlue   = "\033[34m"
	fgRed    = "\033[31m"

	symCheck = "✔"
	symCross = "✖"
)

var (
	forceColor   bool
	disableColor bool
)

func SetColorForcing(force, disable bool) {
	forceColor = force
	disableColor = disable
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func C(color, s string) string {
	if disableColor {
		return s
	}
	if forceColor || isTTY() {
		return color + s + reset
	}
	return s
}

func OK(msg string)   { fmt.Println(C(fgGreen, symCheck+" "+msg)) }
func Fail(msg string) { fmt.Fprintln(os.Stderr, C(fgRed, symCross+" "+msg)) }
