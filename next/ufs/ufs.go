// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

package main

import (
	"log"

	"github.com/rminnich/ninep/next"
)

type ufs struct {
}

func (u *ufs) Rversion(msize uint32, version string) (uint32, string, error) {
	return 8192, "9p2000", nil
}

func NewServer(opts ...next.Opt) (next.NineServer, error) {
	var s = &ufs{}

	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func main() {
	s, err := NewServer()
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("s %v", s)
}
