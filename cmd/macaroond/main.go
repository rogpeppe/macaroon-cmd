package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/juju/loggo"
	"github.com/julienschmidt/httprouter"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
)

var logger = loggo.GetLogger("macaroond")

var (
	netTypeFlag = flag.String("t", "unix", "type of network to listen on (e.g. tcp)")
	addrFlag    = flag.String("addr", "/tmp/macaroond.socket", "address or socket path to listen on")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: macaroond [flags] directory\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
	}
	dir := flag.Arg(0)
	if err := main1(*netTypeFlag, *addrFlag, dir); err != nil {
		log.Fatal(err)
	}
}

func main1(netw string, addr string, dir string) error {
	if _, err := os.Stat(dir); err != nil {
		// TODO create directory?
		return errgo.Mask(err)
	}
	listener, err := net.Listen(netw, addr)
	if err != nil {
		// TODO remove unix socket if it exists and try again
		return errgo.Notef(err, "cannot listen on network %q, addr %q", netw, addr)
	}
	srv := &server{
		dir: dir,
		bakery: bakery.New(bakery.BakeryParams{
			Location: "macaroond",
		}),
	}
	if err := srv.readEncryptedMasterKey(); err != nil {
		return errgo.Notef(err, "cannot read root key file")
	}
	mux := httprouter.New()
	for _, h := range serverParams.Handlers(srv.newHandler) {
		mux.Handle(h.Method, h.Path, h.Handle)
	}
	return http.Serve(listener, mux)
}
