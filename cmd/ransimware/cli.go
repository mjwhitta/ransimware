package main

import (
	"os"
	"strings"

	"gitlab.com/mjwhitta/cli"
	hl "gitlab.com/mjwhitta/hilighter"
	"gitlab.com/mjwhitta/log"
	rw "gitlab.com/mjwhitta/ransimware"
)

// Flags
type cliFlags struct {
	encrypt bool
	exfil   bool
	nocolor bool
	threads int
	verbose bool
	version bool
}

var flags cliFlags

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
			hl.Sprintf("%d: Invalid argument\n", InvalidArgument),
			hl.Sprintf("%d: Missing arguments\n", MissingArguments),
			hl.Sprintf("%d: Extra arguments\n", ExtraArguments),
			hl.Sprintf("%d: Exception", Exception),
		},
		" ",
	)
	cli.Info = strings.Join(
		[]string{
			"Simulate common ransomware behavior and techniques.",
		},
		" ",
	)
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
		false,
		"Exfil simulated data to http://localhost:8080.",
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
		"Use the specified thread-pool size for reading files.",
	)
	cli.Flag(
		&flags.verbose,
		"v",
		"verbose",
		false,
		"Show show stacktrace if error.",
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
