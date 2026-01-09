package ransimware

import _ "embed" // Import embed for the DefaultPNG

// EncryptFunc defines a function pointer that can be used to encrypt
// file contents before exfil.
type EncryptFunc func(fn string, b []byte) ([]byte, error)

// ExfilFunc defines a function pointer that can be used to exil file
// contents.
type ExfilFunc func(fn string, b []byte) error

// NotifyFunc defines a function pointer that can be used to notify
// the user of the ransom.
type NotifyFunc func() error

// Version is the package version
const Version string = "0.30.4"

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

// DefaultPNG is an example PNG for use with WallpaperNotify().
//
//go:embed ransimware.png
var DefaultPNG []byte
