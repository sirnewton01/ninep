// This is a ufs server.
package main

import (
	"flag"
	"log"
	"net"

	"github.com/Harvey-OS/ninep/stub"
	"github.com/Harvey-OS/ninep/filesystem"
)

var (
	ntype = flag.String("ntype", "tcp4", "Default network type")
	naddr = flag.String("addr", ":5640", "Network address")
)

func main() {
	flag.Parse()
	l, err := net.Listen(*ntype, *naddr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("Accept: %v", err)
		}

		_, err = ufs.NewUFS(func(s *stub.Server) error {
			s.FromNet, s.ToNet = c, c
			s.Trace = nil  // log.Printf
			return nil
		})
		if err != nil {
		   log.Printf("Error: %v", err)
		}
	}

}
