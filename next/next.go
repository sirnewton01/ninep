// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

//go:generate go run gen.go -output enc_helpers.go

package next

import (
	"runtime"
	"sync/atomic"
)

var (
	tags chan Tag
	fid uint64
)

func init() {
	tags = make(chan Tag, 1<<16)
	for i := 0; i < 1 << 16; i++ {
		tags <- Tag(i)
	}
}

// GetTag gets a tag to be used to identify a message.
func GetTag() Tag {
	t := <- tags
	runtime.SetFinalizer(&t, func (t *Tag) {
		tags <- *t
	})
	return t
}

// GetFID gets a fid to be used to identify a resource for a 9p client.
// For a given lifetime of a 9p client, FIDS are unique (i.e. not reused as in
// many 9p client libraries).
func GetFID() FID {
	return FID(atomic.AddUint64(&fid, 1))
}
