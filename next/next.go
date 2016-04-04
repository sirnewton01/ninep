// Package next implements the next version of ninep. We use generators, goroutines, and channels,
// and build on what we've learned

//go:generate go run gen.go -output enc_helpers.go

package next

import (
	"bytes"
	"io"
	"log"
	"runtime"
	"sync/atomic"
)

type Opt func(...interface{}) error

var (
	tags chan Tag
	fid  uint64
)

func init() {
	tags = make(chan Tag, 1<<16)
	for i := 0; i < 1<<16; i++ {
		tags <- Tag(i)
	}
}

// GetTag gets a tag to be used to identify a message.
func GetTag() Tag {
	t := <-tags
	runtime.SetFinalizer(&t, func(t *Tag) {
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

func (c *Client) readServerPackets() {
	defer c.Server.Close()
	for !c.Dead {
		l := make([]byte, 64)
		if n, err := c.Server.Read(l); err != nil || n < 7 {
			log.Printf("readServerPackets: short read: %v", err)
			c.Dead = true
			return
		}
		s := int64(l[0]) + int64(l[1])<<8 + int64(l[2])<<16 + int64(l[3])<<24
		b := bytes.NewBuffer(l)
		r := io.LimitReader(c.Server, s)
		if _, err := io.Copy(b, r); err != nil {
			log.Printf("readServerPackets: short read: %v", err)
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
			t := <-tags
			r.b[5] = uint8(t)
			r.b[6] = uint8(t >> 8)
			c.RPC[int(t)] = r
			if _, err := c.Server.Write(r.b); err != nil {
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

func NewClient(opts ...Opt) (*Client, error) {
	var c = &Client{}

	c.Tags = make(chan Tag, NumTags-1)
	for i := 0; i < int(NOTAG); i++ {
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
