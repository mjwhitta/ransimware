//go:build !windows

package ransimware

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mjwhitta/errors"
)

// ExecuteScript will run shell commands using the provided method, as
// well as attempt to clean up artifacts, if requested. Supported
// methods include:
//   - #!bash (will write and run script)
//   - #!sh (will write and run script)
//   - #!zsh (will write and run script)
//   - ash
//   - bash
//   - csh
//   - dash
//   - ksh
//   - sh
//   - shell (same as bash)
//   - tcsh
//   - zsh
func ExecuteScript(
	method string,
	clean bool,
	cmds ...string,
) (string, error) {
	if len(cmds) == 0 {
		return "", nil
	}

	switch method {
	case "#!bash":
		return executeShellScript(
			cmds[0],
			append([]string{"#!/usr/bin/env bash"}, cmds[1:]...),
			clean,
		)
	case "#!sh":
		return executeShellScript(
			cmds[0],
			append([]string{"#!/usr/bin/env sh"}, cmds[1:]...),
			clean,
		)
	case "#!zsh":
		return executeShellScript(
			cmds[0],
			append([]string{"#!/usr/bin/env zsh"}, cmds[1:]...),
			clean,
		)
	case "ash":
		return executeShell("ash", cmds)
	case "bash", "shell":
		return executeShell("bash", cmds)
	case "csh":
		return executeShell("csh", cmds)
	case "dash":
		return executeShell("dash", cmds)
	case "ksh":
		return executeShell("ksh", cmds)
	case "sh":
		return executeShell("sh", cmds)
	case "tcsh":
		return executeShell("tcsh", cmds)
	case "zsh":
		return executeShell("zsh", cmds)
	default:
		return "", errors.Newf("unsupported method: %s", method)
	}
}

func executeShell(shell string, cmds []string) (string, error) {
	var e error
	var o []byte
	var out []string

	// Run cmds
	for _, cmd := range cmds {
		//nolint:gosec // G204 - That's kinda the whole point
		if o, e = exec.Command(shell, "-c", cmd).Output(); e != nil {
			e = errors.Newf("command \"%s\" failed: %w", cmd, e)
			return strings.Join(out, "\n"), e
		}

		if len(o) > 0 {
			out = append(out, strings.TrimSpace(string(o)))
		}
	}

	return strings.Join(out, "\n"), nil
}

func executeShellScript(
	name string,
	cmds []string,
	clean bool,
) (_ string, e error) {
	var b []byte

	// Create shell script
	if e = writeScript(name, cmds); e != nil {
		return "", e
	}

	// Run shell script
	b, e = exec.Command(name).Output()

	// Clean up, if requested
	if clean {
		defer func() {
			if e == nil {
				e = os.Remove(name)
			}
		}()
	}

	// Check for error
	if e != nil {
		return "", errors.Newf("command \"%s\" failed: %w", name, e)
	}

	return strings.TrimSpace(string(b)), nil
}

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
// For any OS other than Windows, this is a no-op.
func WallpaperNotify(
	img string,
	png []byte,
	style uint,
	clean bool,
) NotifyFunc {
	return func() error {
		// return errors.New("unsupported OS")
		return nil
	}
}

func writeScript(name string, cmds []string) error {
	//nolint:gosec // G302 - Needs to be executable
	var e error = os.WriteFile(
		filepath.Clean(name),
		[]byte(strings.Join(cmds, "\n")),
		//nolint:mnd // u=rwx,go=-
		0o700,
	)

	if e != nil {
		return errors.Newf("failed to write to %s: %w", name, e)
	}

	return nil
}
