package main

import (
	"encoding/base64"
	"net/http"
	"os"
	"sync"

	ws "github.com/gorilla/websocket"
	"github.com/mjwhitta/cli"
	hl "github.com/mjwhitta/hilighter"
	"github.com/mjwhitta/log"
)

var b64 bool
var bytesPerG float64 = 1024 * 1024 * 1024
var count float64
var m *sync.Mutex
var port uint
var showCount bool

func init() {
	cli.Align = true
	cli.Banner = hl.Sprintf("%s [OPTIONS]", os.Args[0])
	cli.Info("Super simple Websocket listener.")
	cli.Flag(
		&b64,
		"b",
		"b64",
		true,
		"Incoming data is base64 encoded (default: true).",
	)
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
	mux.HandleFunc("/", wsHandler)

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

func wsHandler(w http.ResponseWriter, req *http.Request) {
	var b []byte
	var c *ws.Conn
	var e error
	var upgrader = ws.Upgrader{}
	var tmp int

	if c, e = upgrader.Upgrade(w, req, nil); e != nil {
		log.Err("Websocket create fail: " + e.Error())
		return
	}

	for {
		if _, b, e = c.ReadMessage(); e != nil {
			if !ws.IsCloseError(e, ws.CloseNormalClosure) {
				log.Err("Websocket read fail: " + e.Error())
			}

			break
		}

		if showCount {
			if len(b) > 0 {
				tmp = len(b)

				if b64 {
					tmp = base64.StdEncoding.DecodedLen(tmp)
				}

				m.Lock()
				count += float64(tmp) / bytesPerG // In GBs
				m.Unlock()
			}

			hl.Printf("\x1b[1A%f GB\n", count)
		} else {
			log.Good(string(b))
		}
	}
}
