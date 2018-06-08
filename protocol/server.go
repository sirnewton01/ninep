// Copyright 2012 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// This code is imported from the old ninep repo,
// with some changes.

package protocol

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
)

// Server is a 9p server.
// For now it's extremely serial. But we will use a chan for replies to ensure that
// we can go to a more concurrent one later.
type Server struct {
	NS        NineServer
	D         Dispatcher
	Versioned bool
	FromNet   io.ReadCloser
	ToNet     io.WriteCloser
	Replies   chan RPCReply
	Trace     Tracer
	Dead      bool

	fprofile *os.File
}

func NewServer(ns NineServer, opts ...ServerOpt) (*Server, error) {
	s := &Server{}
	s.Replies = make(chan RPCReply, NumTags)
	s.NS = ns
	s.D = Dispatch
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) String() string {
	return fmt.Sprintf("Versioned %v Dead %v %d replies pending", s.Versioned, s.Dead, len(s.Replies))
}

func (s *Server) beginSrvProfile() {
	var err error
	s.fprofile, err = ioutil.TempFile(filepath.Dir(*serverprofile), filepath.Base(*serverprofile))
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(s.fprofile)
}

func (s *Server) endSrvProfile() {
	pprof.StopCPUProfile()
	s.fprofile.Close()
	log.Println("writing cpuprofile to", s.fprofile.Name())
}

func (s *Server) readNetPackets() {
	if s.FromNet == nil {
		s.Dead = true
		return
	}
	defer s.FromNet.Close()
	defer s.ToNet.Close()
	if s.Trace != nil {
		s.Trace("Starting readNetPackets")
	}
	if *serverprofile != "" {
		s.beginSrvProfile()
		defer s.endSrvProfile()
	}
	for !s.Dead {
		l := make([]byte, 7)
		if n, err := s.FromNet.Read(l); err != nil || n < 7 {
			log.Printf("readNetPackets: short read: %v", err)
			s.Dead = true
			return
		}
		sz := int64(l[0]) + int64(l[1])<<8 + int64(l[2])<<16 + int64(l[3])<<24
		t := MType(l[4])
		b := bytes.NewBuffer(l[5:])
		r := io.LimitReader(s.FromNet, sz-7)
		if _, err := io.Copy(b, r); err != nil {
			log.Printf("readNetPackets: short read: %v", err)
			s.Dead = true
			return
		}
		if s.Trace != nil {
			s.Trace("readNetPackets: got %v, len %d, sending to IO", RPCNames[MType(l[4])], b.Len())
		}
		//panic(fmt.Sprintf("packet is %v", b.Bytes()[:]))
		//panic(fmt.Sprintf("s is %v", s))
		if err := s.D(s, b, t); err != nil {
			log.Printf("%v: %v", RPCNames[MType(l[4])], err)
		}
		if s.Trace != nil {
			s.Trace("readNetPackets: Write %v back", b)
		}
		amt, err := s.ToNet.Write(b.Bytes())
		if err != nil {
			log.Printf("readNetPackets: write error: %v", err)
			s.Dead = true
			return
		}
		if s.Trace != nil {
			s.Trace("Returned %v amt %v", b, amt)
		}
	}
}

func (s *Server) Start() {
	go s.readNetPackets()
}

func (s *Server) NineServer() NineServer {
	return s.NS
}

// Dispatch dispatches request to different functions.
// It's also the the first place we try to establish server semantics.
// We could do this with interface assertions and such a la rsc/fuse
// but most people I talked do disliked that. So we don't. If you want
// to make things optional, just define the ones you want to implement in this case.
func Dispatch(s *Server, b *bytes.Buffer, t MType) error {
	switch t {
	case Tversion:
		s.Versioned = true
	default:
		if !s.Versioned {
			m := fmt.Sprintf("Dispatch: %v not allowed before Tversion", RPCNames[t])
			// Yuck. Provide helper.
			d := b.Bytes()
			MarshalRerrorPkt(b, Tag(d[0])|Tag(d[1])<<8, m)
			return fmt.Errorf("Dispatch: %v not allowed before Tversion", RPCNames[t])
		}
	}
	switch t {
	case Tversion:
		return s.SrvRversion(b)
	case Tattach:
		return s.SrvRattach(b)
	case Tflush:
		return s.SrvRflush(b)
	case Twalk:
		return s.SrvRwalk(b)
	case Topen:
		return s.SrvRopen(b)
	case Tcreate:
		return s.SrvRcreate(b)
	case Tclunk:
		return s.SrvRclunk(b)
	case Tstat:
		return s.SrvRstat(b)
	case Twstat:
		return s.SrvRwstat(b)
	case Tremove:
		return s.SrvRremove(b)
	case Tread:
		return s.SrvRread(b)
	case Twrite:
		return s.SrvRwrite(b)
	}
	// This has been tested by removing Attach from the switch.
	ServerError(b, fmt.Sprintf("Dispatch: %v not supported", RPCNames[t]))
	return nil
}
