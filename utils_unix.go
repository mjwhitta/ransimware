//+build !windows

package ransimware

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultNotify is the default notify behavior.
var DefaultNotify = func() error {
	var e error
	var f *os.File
	var home string

	if home, e = os.UserHomeDir(); e != nil {
		return e
	}

	f, e = os.Create(filepath.Join(home, "ransomnote.txt"))
	if e != nil {
		return e
	}
	defer f.Close()

	f.WriteString("This is a ransomware simulation.\n")

	return nil
}

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(image string, png []byte) NotifyFunc {
	return func() error {
		return fmt.Errorf("OS not supported")
	}
}
