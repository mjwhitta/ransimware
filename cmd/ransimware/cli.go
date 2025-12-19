package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mjwhitta/cli"
	hl "github.com/mjwhitta/hilighter"
	"github.com/mjwhitta/log"
	rw "github.com/mjwhitta/ransimware"
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
	encrypt   string
	exfil     string
	names     bool
	nocolor   bool
	note      string
	threads   int
	threshold uint64
	verbose   bool
	version   bool
	waitEvery int64
	waitFor   int64
	wallpaper bool
}

func init() {
	// Configure cli package
	cli.Align = true
	cli.Authors = []string{"Miles Whittaker <mj@whitta.dev>"}
	cli.Banner = "" +
		filepath.Base(os.Args[0]) + " [OPTIONS] [dir1]... [dirN]"
	cli.BugEmail = "ransimware.bugs@whitta.dev"

	cli.ExitStatus(
		"Normally the exit status is 0. In the event of an error the",
		"exit status will be one of the below:\n\n",
		fmt.Sprintf("%d: Invalid option\n", InvalidOption),
		fmt.Sprintf("%d: Missing option\n", MissingOption),
		fmt.Sprintf("%d: Invalid argument\n", InvalidArgument),
		fmt.Sprintf("%d: Missing argument\n", MissingArgument),
		fmt.Sprintf("%d: Extra argument\n", ExtraArgument),
		fmt.Sprintf("%d: Exception", Exception),
	)
	cli.Info("Simulate common ransomware behavior and techniques.")

	cli.Title = "Ransimware"

	// Parse cli flags
	cli.Flag(
		&flags.encrypt,
		"e",
		"encrypt",
		"",
		"Use specified password to simulate encryption of file",
		"contents using AES (default: no encryption).",
	)
	cli.Flag(
		&flags.exfil,
		"x",
		"exfil",
		"",
		"Exfil simulated data to specified location",
		"(default: no exfil, supports: dns, ftp, http(s), ws(s)).",
	)
	cli.Flag(
		&flags.names,
		"n",
		"names",
		false,
		"Exfil filenames as well.",
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
		8, //nolint:mnd // 8 threads
		"Use specified thread pool size for reading files",
		"(default: 8).",
	)
	cli.Flag(
		&flags.threshold,
		"threshold",
		0,
		"Stop exfil after specified bytes has been exceeded",
		"(default: 0 = all).",
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
		"Wait every specified number of seconds",
		"(default: 0 = no wait). Requires --wait-for.",
	)
	cli.Flag(
		&flags.waitFor,
		"wait-for",
		0,
		"Wait for the specified number of seconds",
		"(default: 0 = no wait). Requires --wait-every.",
	)
	cli.Flag(&flags.version, "V", "version", false, "Show version.")

	switch runtime.GOOS {
	case "windows":
		cli.Flag(
			&flags.note,
			"note",
			"",
			"Leave a ransom note on the user's Desktop.",
		)
		cli.Flag(
			&flags.wallpaper,
			"w",
			"wallpaper",
			false,
			"Change the user's desktop wallpaper.",
		)
	default:
		cli.Flag(
			&flags.note,
			"note",
			"",
			"Leave a ransom note in the user's home directory.",
		)
	}

	cli.Parse()
}

// Process cli flags and ensure no issues
func validate() {
	hl.Disable(flags.nocolor)

	// Short circuit if version was requested
	if flags.version {
		fmt.Println(
			filepath.Base(os.Args[0]) + " version " + rw.Version,
		)
		os.Exit(Good)
	}

	if strings.TrimSpace(flags.note) == "" {
		flags.note = ""
	}

	switch {
	case flags.threads <= 0:
		log.ErrX(InvalidArgument, "--threads must be > 0")
	case flags.waitEvery < 0:
		log.ErrX(InvalidArgument, "--wait-every must be >= 0")
	case flags.waitFor < 0:
		log.ErrX(InvalidArgument, "--wait-for must be >= 0")
	case (flags.waitEvery > 0) && (flags.waitFor == 0):
		log.ErrX(InvalidArgument, "--wait-every requires --wait-for")
	case (flags.waitEvery == 0) && (flags.waitFor > 0):
		log.ErrX(InvalidArgument, "--wait-for requires --wait-every")
	}
}
