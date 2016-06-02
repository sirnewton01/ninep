// Package protocol implements the 9p protocol using the stubs.

package protocol

import (
	"bytes"
	"fmt"

	"github.com/rminnich/ninep/stub"
)

func NewServer(ns stub.NineServer, opts ...stub.ServerOpt) (*stub.Server, error) {
	s := &stub.Server{}
	s.Replies = make(chan stub.RPCReply, stub.NumTags)
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
func Dispatch(s *stub.Server, b *bytes.Buffer, t stub.MType) error {
	switch t {
	case stub.Tversion:
		s.Versioned = true
	default:
		if !s.Versioned {
			m := fmt.Sprintf("Dispatch: %v not allowed before Tversion", stub.RPCNames[t])
			// Yuck. Provide helper.
			d := b.Bytes()
			stub.MarshalRerrorPkt(b, stub.Tag(d[0])|stub.Tag(d[1]<<8), m)
			return fmt.Errorf("Dispatch: %v not allowed before Tversion", stub.RPCNames[t])
		}
	}
	switch t {
	case stub.Tversion:
		return s.SrvRversion(b)
	case stub.Tattach:
		return s.SrvRattach(b)
	case stub.Tflush:
		return s.SrvRflush(b)
	case stub.Twalk:
		return s.SrvRwalk(b)
	case stub.Topen:
		return s.SrvRopen(b)
	case stub.Tcreate:
		return s.SrvRcreate(b)
	case stub.Tclunk:
		return s.SrvRclunk(b)
	case stub.Tstat:
		return s.SrvRstat(b)
	case stub.Twstat:
		return s.SrvRwstat(b)
	case stub.Tremove:
		return s.SrvRremove(b)
	case stub.Tread:
		return s.SrvRread(b)
	case stub.Twrite:
		return s.SrvRwrite(b)
	}
	// This has been tested by removing Attach from the switch.
	stub.ServerError(b, fmt.Sprintf("Dispatch: %v not supported", stub.RPCNames[t]))
	return nil
}
