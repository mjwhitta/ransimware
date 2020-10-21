package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"gitlab.com/mjwhitta/log"
)

var m = &sync.Mutex{}

func handler(w http.ResponseWriter, req *http.Request) {
	var body []byte
	var e error

	if body, e = ioutil.ReadAll(req.Body); e != nil {
		log.Err(e.Error())
		return
	}

	m.Lock()
	log.SubInfof("%s %s", req.Method, req.URL.String())
	for k, v := range req.Header {
		log.Warnf("%s: %s", k, strings.Join(v, "; "))
	}
	if len(body) > 0 {
		log.Good(string(body))
	}
	m.Unlock()

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
