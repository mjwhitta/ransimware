//go:build unix

package ransimware

import (
	"os"
	"os/exec"
	"strings"

	"github.com/mjwhitta/errors"
)

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
) (string, error) {
	var e error
	var o []byte

	// Create shell script
	if e = writeScript(name, cmds); e != nil {
		return "", e
	}

	// Make script executable
	if e = os.Chmod(name, os.ModePerm); e != nil {
		e = errors.Newf("failed to make script executable: %w", e)
		return "", e
	}

	// Run shell script
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

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(
	img string,
	png []byte,
	fit string,
	clean bool,
) NotifyFunc {
	return func() error {
		return errors.New("unsupported OS")
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
