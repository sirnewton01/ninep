package ufs

import (
	"io"
	"testing"

	"github.com/rminnich/ninep/rpc"
)

func TestNew(t *testing.T) {
	n, err := NewUFS()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("n is %v", n)
}

func TestMount(t *testing.T) {
	sr, cw := io.Pipe()
	cr, sw := io.Pipe()


	c, err := rpc.NewClient(func(c *rpc.Client) error {
		c.FromNet, c.ToNet = cr, cw
		return nil
	},
		func(c *rpc.Client) error {
			c.Msize = 8192
			c.Trace = t.Logf
			return nil
		})
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("Client is %v", c.String())

	n, err := NewUFS(func(s *rpc.Server) error {
		s.FromNet, s.ToNet = sr, sw
		s.Trace = t.Logf
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("n is %v", n)


	
}
