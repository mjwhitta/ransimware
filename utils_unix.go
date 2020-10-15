//+build !windows

package ransimware

import "fmt"

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(image string, png []byte) NotifyFunc {
	return func() error {
		return fmt.Errorf("OS not supported")
	}
}
