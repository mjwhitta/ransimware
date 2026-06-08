package ransimware

import _ "embed" // Import embed for the DefaultPNG

type (
	// EncryptFunc defines a function pointer that can be used to
	// encrypt file contents before exfil.
	EncryptFunc func(fn string, b []byte) ([]byte, error)
	// ExfilFunc defines a function pointer that can be used to exil
	// file contents.
	ExfilFunc func(fn string, b []byte) error
	// NotifyFunc defines a function pointer that can be used to
	// notify the user of the ransom.
	NotifyFunc func() error
)

// Version is the package version
const Version string = "0.30.7"

// Desktop wallpaper style consts
//
//nolint:grouper // Separate b/c enum/iota
const (
	WallpaperStyleCenter uint = iota
	WallpaperStyleFill
	WallpaperStyleFit
	WallpaperStyleSpan
	WallpaperStyleStretch
	WallpaperStyleTile
)

var (
	// DefaultEncrypt is the default encryption behavior.
	DefaultEncrypt = func(path string, b []byte) ([]byte, error) {
		return b, nil
	}

	// DefaultExfil is the default exfil behavior.
	DefaultExfil = func(path string, b []byte) error {
		return nil
	}

	// DefaultNotify is the default notify behavior.
	DefaultNotify = func() error {
		return nil
	}

	// DefaultPNG is an example PNG for use with WallpaperNotify().
	//
	//go:embed ransimware.png
	DefaultPNG []byte
)
