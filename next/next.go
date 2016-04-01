// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

//go:generate go run gen.go -output enc_helpers.go

package next

import (
	"runtime"
)

var (
	tags chan Tag
)

func init() {
	tags = make(chan Tag, 1<<16)
	for i := 0; i < 1 << 16; i++ {
		tags <- Tag(i)
	}
}

func GetTag() Tag {
	t := <- tags
	runtime.SetFinalizer(&t, func (t *Tag) {
		tags <- *t
	})
	return t
}
