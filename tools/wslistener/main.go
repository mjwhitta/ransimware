package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"

	ws "github.com/gorilla/websocket"
	"gitlab.com/mjwhitta/log"
)

func init() {
	flag.Parse()
}

func main() {
	var addr string
	var e error
	var mux *http.ServeMux
	var port int64 = 8080
	var server *http.Server

	if flag.NArg() > 0 {
		if port, e = strconv.ParseInt(flag.Arg(0), 10, 64); e != nil {
			panic(e)
		}
	}

	addr = fmt.Sprintf("0.0.0.0:%d", port)

	mux = http.NewServeMux()
	mux.HandleFunc("/", wsHandler)

	server = &http.Server{Addr: addr, Handler: mux}

	log.Infof("Listening on %s", addr)
	e = server.ListenAndServe()

	switch e {
	case http.ErrServerClosed:
	default:
		panic(e)
	}
}

func wsHandler(w http.ResponseWriter, req *http.Request) {
	var body []byte
	var c *ws.Conn
	var e error
	var upgrader = ws.Upgrader{}

	if c, e = upgrader.Upgrade(w, req, nil); e != nil {
		log.Err("Websocket create fail: " + e.Error())
		return
	}

	for {
		if _, body, e = c.ReadMessage(); e != nil {
			if !ws.IsCloseError(e, ws.CloseNormalClosure) {
				log.Err("Websocket read fail: " + e.Error())
			}

			break
		}

		fmt.Println(string(body))
	}
}
