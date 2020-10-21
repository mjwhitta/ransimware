//+build !windows

package ransimware

import (
	"fmt"
	"net/http"
	"net/url"
)

func getSystemProxy() func(req *http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment
}

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(image string, png []byte) NotifyFunc {
	return func() error {
		return fmt.Errorf("OS not supported")
	}
}
