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
	defer c.FromNet.Close()
	for !c.Dead {
		l := make([]byte, 64)
		if n, err := c.FromNet.Read(l); err != nil || n < 7 {
			log.Printf("readNetPackets: short read: %v", err)
			c.Dead = true
			return
		}
		s := int64(l[0]) + int64(l[1])<<8 + int64(l[2])<<16 + int64(l[3])<<24
		b := bytes.NewBuffer(l)
		r := io.LimitReader(c.FromNet, s)
		if _, err := io.Copy(b, r); err != nil {
			log.Printf("readNetPackets: short read: %v", err)
			c.Dead = true
			return
		}
		c.FromServer <- &RPCReply{b: b.Bytes()}
	}

}

func (c *Client) IO() {
	for {
		select {
		case r := <-c.FromClient:
			t := <-c.Tags
			r.b[5] = uint8(t)
			r.b[6] = uint8(t >> 8)
			c.RPC[int(t)] = r
			if _, err := c.ToNet.Write(r.b); err != nil {
				c.Dead = true
				log.Printf("Write to server: %v", err)
				return
			}
		case r := <-c.FromServer:
			t := r.b[5] + r.b[6]<<8
			c.RPC[t].Reply <- r.b
		}
	}
}

func (c *Client) String() string{
	z := map[bool]string {false: "Alive", true: "Dead"}
	return fmt.Sprintf("%v tags available, Msize %v, %v", len(c.Tags), c.Msize, z[c.Dead])
}

func NewClient(opts ...ClientOpt) (*Client, error) {
	var c = &Client{}

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
	go c.IO()
	return c, nil
}

func NewServer(opts ...ServerOpt) (*Server, error) {
	s := &Server{}
	s.Replies = make(chan RPCReply, NumTags)
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}
