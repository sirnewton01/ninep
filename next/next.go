// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

//go:generate go run gen.go -output enc_helpers.go

package next

import (
	"bytes"
	"fmt"

	"github.com/rminnich/ninep/rpc"
)

func NewServer(ns rpc.NineServer, opts ...rpc.ServerOpt) (*rpc.Server, error) {
	s := &rpc.Server{}
	s.Replies = make(chan rpc.RPCReply, rpc.NumTags)
	s.NS = ns
	s.D = Dispatch
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}
// Dispatch dispatches request to different functions.
// It's also the the first place we try to establish server semantics.
// We could do this with interface assertions and such a la rsc/fuse
// but most people I talked do disliked that. So we don't. If you want
// to make things optional, just define the ones you want to implement in this case.
func Dispatch(s *rpc.Server, b *bytes.Buffer, t rpc.MType) error {
	switch t {
	case rpc.Tversion:
		s.Versioned = true
	default:
		if !s.Versioned {
			m := fmt.Sprintf("Dispatch: %v not allowed before Tversion", rpc.RPCNames[t])
			// Yuck. Provide helper.
			d := b.Bytes()
			rpc.MarshalRerrorPkt(b, rpc.Tag(d[0])|rpc.Tag(d[1]<<8), m)
			return fmt.Errorf("Dispatch: %v not allowed before Tversion", rpc.RPCNames[t])
		}
	}
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
	case rpc.Tcreate:
		return s.SrvRcreate(b)
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

