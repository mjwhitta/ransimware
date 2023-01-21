package main

import (
	"net/http"
	"net/http/httputil"
	"os"
	"sync"

	"github.com/mjwhitta/cli"
	hl "github.com/mjwhitta/hilighter"
	"github.com/mjwhitta/log"
)

var count float64
var m *sync.Mutex
var port uint
var showCount bool

func handler(w http.ResponseWriter, req *http.Request) {
	var b []byte
	var e error

	if showCount {
		if req.ContentLength > 0 {
			m.Lock()
			// In GBs
			count += float64(req.ContentLength) / (1024 * 1024 * 1024)
			m.Unlock()
		}

		hl.Printf("\x1b[1A%f GB\n", count)
	} else {
		if b, e = httputil.DumpRequest(req, true); e != nil {
			log.Err(e.Error())
			return
		}

		log.Good(string(b))
	}

	w.Write([]byte("Success"))
}

func init() {
	cli.Align = true
	cli.Banner = hl.Sprintf("%s [OPTIONS]", os.Args[0])
	cli.Info = "Super simple HTTP listener."
	cli.Flag(&showCount, "c", "count", false, "Show running count.")
	cli.Flag(&port, "p", "port", 8080, "Listen on specified port.")
	cli.Parse()

	m = &sync.Mutex{}
}

func main() {
	var addr string
	var e error
	var mux *http.ServeMux
	var server *http.Server

	addr = hl.Sprintf("0.0.0.0:%d", port)

	mux = http.NewServeMux()
	mux.HandleFunc("/", handler)

	server = &http.Server{Addr: addr, Handler: mux}

	log.Infof("Listening on %s", addr)
	if showCount {
		hl.Printf("%f GB\n", count)
	}
	e = server.ListenAndServe()

	switch e {
	case http.ErrServerClosed:
	default:
		panic(e)
	}
}
