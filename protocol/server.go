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
	"log"
	"net"
	"time"
)

const DefaultAddr = ":5640"

// Server is a 9p server.
// For now it's extremely serial. But we will use a chan for replies to ensure that
// we can go to a more concurrent one later.
type Server struct {
	NS NineServer
	D  Dispatcher

	// TCP address to listen on, default is DefaultAddr
	Addr string

	// Trace function for logging
	Trace Tracer
}

type conn struct {
	// server on which the connection arrived.
	server *Server

	// rwc is the underlying network connection.
	rwc net.Conn

	// remoteAddr is rwc.RemoteAddr().String(). See note in net/http/server.go.
	remoteAddr string

	// versioned set to true after first Tversion
	versioned bool

	// replies
	replies chan RPCReply

	// dead is set to true when we finish reading packets.
	dead bool
}

func NewServer(ns NineServer, opts ...ServerOpt) (*Server, error) {
	s := &Server{}
	s.NS = ns
	s.D = Dispatch
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) newConn(rwc net.Conn) *conn {
	c := &conn{
		server:  s,
		rwc:     rwc,
		replies: make(chan RPCReply, NumTags),
	}

	return c
}

// ListenAndServe starts a new Listener on e.Addr and then calls serve.
func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = DefaultAddr
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

// Serve accepts incoming connections on the Listener and calls e.Accept on
// each connection.
func (s *Server) Serve(ln net.Listener) error {
	defer ln.Close()

	var tempDelay time.Duration // how long to sleep on accept failure

	// from http.Server.Serve
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				s.logf("ufs: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		if err := s.Accept(conn); err != nil {
			return err
		}
	}
}

// Accept a new connection, typically called via Serve but may be called
// directly if there's a connection from an exotic listener.
func (s *Server) Accept(conn net.Conn) error {
	c := s.newConn(conn)

	go c.serve()
	return nil
}

func (s *Server) String() string {
	// TODO
	return ""
}

func (s *Server) logf(format string, args ...interface{}) {
	if s.Trace != nil {
		s.Trace(format, args...)
	}
}

func (c *conn) String() string {
	return fmt.Sprintf("Versioned %v Dead %v %d replies pending", c.versioned, c.dead, len(c.replies))
}

func (c *conn) logf(format string, args ...interface{}) {
	// prepend some info about the conn
	c.server.logf("[%v] "+format, append([]interface{}{c.remoteAddr}, args)...)
}

func (c *conn) serve() {
	if c.rwc == nil {
		c.dead = true
		return
	}

	c.remoteAddr = c.rwc.RemoteAddr().String()

	defer c.rwc.Close()

	c.logf("Starting readNetPackets")

	for !c.dead {
		l := make([]byte, 7)
		if n, err := c.rwc.Read(l); err != nil || n < 7 {
			log.Printf("readNetPackets: short read: %v", err)
			c.dead = true
			return
		}
		sz := int64(l[0]) + int64(l[1])<<8 + int64(l[2])<<16 + int64(l[3])<<24
		t := MType(l[4])
		b := bytes.NewBuffer(l[5:])
		r := io.LimitReader(c.rwc, sz-7)
		if _, err := io.Copy(b, r); err != nil {
			log.Printf("readNetPackets: short read: %v", err)
			c.dead = true
			return
		}
		c.logf("readNetPackets: got %v, len %d, sending to IO", RPCNames[MType(l[4])], b.Len())
		//panic(fmt.Sprintf("packet is %v", b.Bytes()[:]))
		//panic(fmt.Sprintf("s is %v", s))
		if err := c.server.D(c.server, b, t); err != nil {
			log.Printf("%v: %v", RPCNames[MType(l[4])], err)
		}
		c.logf("readNetPackets: Write %v back", b)
		amt, err := c.rwc.Write(b.Bytes())
		if err != nil {
			log.Printf("readNetPackets: write error: %v", err)
			c.dead = true
			return
		}
		c.logf("Returned %v amt %v", b, amt)
	}
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
	// TODO: should this be in c.serve?
	/*
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
	*/
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
