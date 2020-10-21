package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strconv"

	"gitlab.com/mjwhitta/log"
)

func handler(w http.ResponseWriter, req *http.Request) {
	var dump []byte
	var e error

	if dump, e = httputil.DumpRequest(req, true); e != nil {
		log.Err(e.Error())
		return
	}

	log.Good(string(dump))

	w.Write([]byte("Success"))
}

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
	mux.HandleFunc("/", handler)

	server = &http.Server{Addr: addr, Handler: mux}

	log.Infof("Listening on %s", addr)
	e = server.ListenAndServe()

	switch e {
	case http.ErrServerClosed:
	default:
		panic(e)
	}
}
