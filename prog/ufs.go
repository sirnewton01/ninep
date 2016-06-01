// This is a ufs server.
package main

import (
	"flag"
	"log"
	"net"

	"github.com/rminnich/ninep/rpc"
	"github.com/rminnich/ninep/ufs"
)

var (
	ntype = flag.String("ntype", "tcp4", "Default network type")
	naddr = flag.String("addr", ":5640", "Network address")
)

func main() {

	l, err := net.Listen(*ntype, *naddr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("Accept: %v", err)
		}

		_, err = ufs.NewUFS(func(s *rpc.Server) error {
			s.FromNet, s.ToNet = c, c
			s.Trace = log.Printf
			return nil
		})
		if err != nil {
		   log.Printf("Error: %v", err)
		}
	}

}
