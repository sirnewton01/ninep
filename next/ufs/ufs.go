// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

package main

import (
	"log"

	"github.com/rminnich/ninep/next"
)

type ufs struct {
	*next.Server
}

func (u *ufs) Rversion(msize uint32, version string) (uint32, string, error) {
	return 8192, "9p2000", nil
}

func main() {
	s, err := next.NewServer()
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("s %v", s)
}
