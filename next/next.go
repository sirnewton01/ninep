// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

//go:generate go run gen.go -output enc_helpers.go

package next

import (
	"bytes"
	"fmt"

	rpc "github.com/rminnich/ninep/rpc"
)

// Dispatch dispatches request to different functions.
// We could do this with interface assertions and such a la rsc/fuse
// but most people I talked do disliked that. So we don't. If you want
// to make things optional, just define the ones you want to implement in this case.
func (s Server) Dispatch(b *bytes.Buffer, t rpc.MType) error {
	switch t {
	case rpc.Tversion:
		return s.SrvRversion(b)
	case rpc.Tattach:
		return s.SrvRattach(b)
	case rpc.Tflush:
		return s.SrvRflush(b)
	case rpc.Twalk:
		return s.SrvRwalk(b)
	case rpc.Topen:
		return s.SrvRopen(b)
	case rpc.Tclunk:
		return s.SrvRclunk(b)
	case rpc.Tstat:
		return s.SrvRstat(b)
	case rpc.Twstat:
		return s.SrvRwstat(b)
	case rpc.Tremove:
		return s.SrvRremove(b)
	case rpc.Tread:
		return s.SrvRread(b)
	case rpc.Twrite:
		return s.SrvRwrite(b)
	}
	// This has been tested by removing Attach from the switch.
	rpc.ServerError(b, fmt.Sprintf("Dispatch: %v not supported", rpc.RPCNames[t]))
	return nil
}

