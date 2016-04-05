// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

//go:generate go run gen.go -output enc_helpers.go

package next

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"runtime"
	"sync/atomic"
)

var (
	RPCNames = map[MType]string{
		Tversion: "Tversion",
		Rversion: "Rversion",
		Tauth:    "Tauth",
		Rauth:    "Rauth",
		Tattach:  "Tattach",
		Rattach:  "Rattach",
		Terror:   "Terror",
		Rerror:   "Rerror",
		Tflush:   "Tflush",
		Rflush:   "Rflush",
		Twalk:    "Twalk",
		Rwalk:    "Rwalk",
		Topen:    "Topen",
		Ropen:    "Ropen",
		Tcreate:  "Tcreate",
		Rcreate:  "Rcreate",
		Tread:    "Tread",
		Rread:    "Rread",
		Twrite:   "Twrite",
		Rwrite:   "Rwrite",
		Tclunk:   "Tclunk",
		Rclunk:   "Rclunk",
		Tremove:  "Tremove",
		Rremove:  "Rremove",
		Tstat:    "Tstat",
		Rstat:    "Rstat",
		Twstat:   "Twstat",
		Rwstat:   "Rwstat",
	}
)

func noTrace(string, ...interface{}) {}

// GetTag gets a tag to be used to identify a message.
func (c *Client) GetTag() Tag {
	t := <-c.Tags
	runtime.SetFinalizer(&t, func(t *Tag) {
		c.Tags <- *t
	})
	return t
}

// GetFID gets a fid to be used to identify a resource for a 9p client.
// For a given lifetime of a 9p client, FIDS are unique (i.e. not reused as in
// many 9p client libraries).
func (c *Client) GetFID() FID {
	return FID(atomic.AddUint64(&c.FID, 1))
}

func (c *Client) readNetPackets() {
	if c.FromNet == nil {
		c.Dead = true
		return
	}
	defer c.FromNet.Close()
	defer close(c.FromServer)
	c.Trace("Starting readNetPackets")
	for !c.Dead {
		l := make([]byte, 7)
		if n, err := c.FromNet.Read(l); err != nil || n < 7 {
			log.Printf("readNetPackets: short read: %v", err)
			c.Dead = true
			return
		}
		s := int64(l[0]) + int64(l[1])<<8 + int64(l[2])<<16 + int64(l[3])<<24
		b := bytes.NewBuffer(l)
		r := io.LimitReader(c.FromNet, s-7)
		if _, err := io.Copy(b, r); err != nil {
			log.Printf("readNetPackets: short read: %v", err)
			c.Dead = true
			return
		}
		c.Trace("readNetPackets: got %v, len %d, sending to IO", RPCNames[MType(l[4])], b.Len())
		c.FromServer <- &RPCReply{b: b.Bytes()}
	}

}

func (c *Client) IO() {
	go func() {
		for {
			r := <-c.FromClient
			t := <-c.Tags
			r.b[5] = uint8(t)
			r.b[6] = uint8(t >> 8)
			//panic(fmt.Sprintf("Tag for request is %v", t))
			c.RPC[int(t)] = r
			c.Trace("Write %v to ToNet", r.b)
			if _, err := c.ToNet.Write(r.b); err != nil {
				c.Dead = true
				log.Fatalf("Write to server: %v", err)
				return
			}
		}
	}()

	for {
		r := <-c.FromServer
		c.Trace("Read %v FromServer", r.b)
		t := r.b[5] + r.b[6]<<8
		//panic(fmt.Sprintf("Tag for reply is %v", t))
		c.RPC[t].Reply <- r.b
	}
}

func (c *Client) String() string {
	z := map[bool]string{false: "Alive", true: "Dead"}
	return fmt.Sprintf("%v tags available, Msize %v, %v", len(c.Tags), c.Msize, z[c.Dead])
}

func (s *Server) String() string {
	return fmt.Sprintf("%d replies pending", len(s.Replies))
}

func NewClient(opts ...ClientOpt) (*Client, error) {
	var c = &Client{Trace: noTrace}

	c.Tags = make(chan Tag, NumTags)
	for i := 1; i < int(NOTAG); i++ {
		c.Tags <- Tag(i)
	}
	c.FID = 1
	c.RPC = make([]*RPCCall, NumTags)
	for _, o := range opts {
		if err := o(c); err != nil {
			return nil, err
		}
	}
	c.FromClient = make(chan *RPCCall)
	c.FromServer = make(chan *RPCReply)
	go c.IO()
	go c.readNetPackets()
	return c, nil
}

func (s *Server) readNetPackets() {
	if s.FromNet == nil {
		s.Dead = true
		return
	}
	defer s.FromNet.Close()
	defer s.ToNet.Close()
	s.Trace("Starting readNetPackets")
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
		s.Trace("readNetPackets: got %v, len %d, sending to IO", RPCNames[MType(l[4])], b.Len())
		//panic(fmt.Sprintf("packet is %v", b.Bytes()[:]))
		if err := s.NS.Dispatch(b, t); err != nil {
			log.Printf("%v: %v", RPCNames[MType(l[4])], err)
			continue
		}
		s.Trace("readNetPackets: Write %v back", b)
		amt, err := s.ToNet.Write(b.Bytes())
		if err != nil {
			log.Printf("readNetPackets: write error: %v", err)
			s.Dead = true
			return
		}
		s.Trace("Returned %v amt %v", b, amt)
	}

}

func (s *Server) Start() {
	go s.readNetPackets()
}

func NewServer(opts ...ServerOpt) (*Server, error) {
	s := &Server{Trace: noTrace}
	s.Replies = make(chan RPCReply, NumTags)
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}
