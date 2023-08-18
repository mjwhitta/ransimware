package ransimware

import _ "embed" // Import embed for the DefaultPNG

// DefaultPNG is an example PNG for use with WallpaperNotify().
//
//go:embed ransimware.png
var DefaultPNG []byte

// Desktop wallpaper style consts
const (
	DesktopCenter  string = "0"
	DesktopFill    string = "10"
	DesktopFit     string = "6"
	DesktopSpan    string = "22"
	DesktopStretch string = "2"
	DesktopTile    string = "0"
)

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
const Version = "0.24.0"
