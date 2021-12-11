package main

import (
	"os"
	"strings"

	"gitlab.com/mjwhitta/cli"
	hl "gitlab.com/mjwhitta/hilighter"
	"gitlab.com/mjwhitta/log"
	rw "gitlab.com/mjwhitta/ransimware"
)

// Exit status
const (
	Good = iota
	InvalidOption
	MissingOption
	InvalidArgument
	MissingArgument
	ExtraArgument
	Exception
)

// Flags
var flags struct {
	encrypt   bool
	exfil     string
	nocolor   bool
	threads   int
	threshold uint64
	verbose   bool
	version   bool
	waitEvery uint
	waitFor   uint
}

func init() {
	// Configure cli package
	cli.Align = true
	cli.Authors = []string{"Miles Whittaker <mj@whitta.dev>"}
	cli.Banner = hl.Sprintf(
		"%s [OPTIONS] [dir1]... [dirN]",
		os.Args[0],
	)
	cli.BugEmail = "ransimware.bugs@whitta.dev"
	cli.ExitStatus = strings.Join(
		[]string{
			"Normally the exit status is 0. In the event of an error",
			"the exit status will be one of the below:\n\n",
			hl.Sprintf("%d: Invalid option\n", InvalidOption),
			hl.Sprintf("%d: Missing option\n", MissingOption),
			hl.Sprintf("%d: Invalid argument\n", InvalidArgument),
			hl.Sprintf("%d: Missing argument\n", MissingArgument),
			hl.Sprintf("%d: Extra argument\n", ExtraArgument),
			hl.Sprintf("%d: Exception", Exception),
		},
		" ",
	)
	cli.Info = "Simulate common ransomware behavior and techniques."
	cli.Title = "Ransimware"

	// Parse cli flags
	cli.Flag(
		&flags.encrypt,
		"e",
		"encrypt",
		false,
		"Simulate encryption of file contents using AES.",
	)
	cli.Flag(
		&flags.exfil,
		"x",
		"exfil",
		"",
		"Exfil simulated data to specified location.",
	)
	cli.Flag(
		&flags.nocolor,
		"no-color",
		false,
		"Disable colorized output.",
	)
	cli.Flag(
		&flags.threads,
		"t",
		"threads",
		32,
		"Use specified thread-pool size for reading files.",
	)
	cli.Flag(
		&flags.threshold,
		"threshold",
		0,
		"Stop exfil after specified bytes has been exceeded.",
	)
	cli.Flag(
		&flags.verbose,
		"v",
		"verbose",
		false,
		"Show stacktrace, if error.",
	)
	cli.Flag(
		&flags.waitEvery,
		"wait-every",
		0,
		"Wait after the specified number of seconds, repeatedly.",
	)
	cli.Flag(
		&flags.waitFor,
		"wait-for",
		0,
		"Wait for the specified number of seconds.",
	)
	cli.Flag(&flags.version, "V", "version", false, "Show version.")
	cli.Parse()
}

// Process cli flags and ensure no issues
func validate() {
	hl.Disable(flags.nocolor)

	// Short circuit if version was requested
	if flags.version {
		hl.Printf("ransimware version %s\n", rw.Version)
		os.Exit(Good)
	}

	if flags.threads <= 0 {
		log.ErrX(InvalidArgument, "Threads must be > 0")
	}
}
