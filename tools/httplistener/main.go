package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mjwhitta/cli"
	"github.com/mjwhitta/log"
)

var (
	b64       bool
	bytesPerG float64 = 1024 * 1024 * 1024
	count     float64
	m         *sync.Mutex
	port      uint
	showCount bool
)

func handler(w http.ResponseWriter, req *http.Request) {
	var b []byte
	var e error
	var tmp int

	if showCount {
		if req.ContentLength > 0 {
			if tmp = int(req.ContentLength); b64 {
				tmp = base64.StdEncoding.DecodedLen(tmp)
			}

			m.Lock()

			count += float64(tmp) / bytesPerG // In GBs

			m.Unlock()
		}

		fmt.Printf("\x1b[1A%f GB\n", count)
	} else {
		if b, e = httputil.DumpRequest(req, true); e != nil {
			log.Err(e.Error())
			return
		}

		log.Good(string(b))
	}

	_, _ = w.Write([]byte("Success"))
}

func init() {
	cli.Align = true
	cli.Banner = filepath.Base(os.Args[0]) + " [OPTIONS]"

	cli.Info("Super simple HTTP listener.")

	cli.Flag(
		&b64,
		"b",
		"b64",
		true,
		"Incoming data is base64 encoded (default: true).",
	)
	cli.Flag(&showCount, "c", "count", false, "Show running count.")
	cli.Flag(
		&port,
		"p",
		"port",
		8080, //nolint:mnd // Default non-privileged HTTP port
		"Listen on specified port (default: 8080).",
	)
	cli.Parse()

	m = &sync.Mutex{}
}

func main() {
	var addr string
	var e error
	var mux *http.ServeMux
	var server *http.Server

	addr = fmt.Sprintf("0.0.0.0:%d", port)

	mux = http.NewServeMux()
	mux.HandleFunc("/", handler)

	server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, //nolint:mnd // 10 secs
	}

	log.Infof("Listening on %s", addr)

	if showCount {
		fmt.Printf("%f GB\n", count)
	}

	e = server.ListenAndServe()

	switch e {
	case nil, http.ErrServerClosed:
	default:
		panic(e)
	}
}
