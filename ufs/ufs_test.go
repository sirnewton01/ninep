package ufs

import (
	"fmt"
	"io"
	"testing"

	"github.com/rminnich/ninep/rpc"
)

func print(f string, args ...interface{}) {
	fmt.Printf(f+"\n", args...)
}

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
			c.Trace = print//t.Logf
			return nil
		})
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("Client is %v", c.String())

	n, err := NewUFS(func(s *rpc.Server) error {
		s.FromNet, s.ToNet = sr, sw
		s.Trace = print//t.Logf
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("n is %v", n)

	m, v, err := c.CallTversion(8000, "9P2000")
	if err != nil {
		t.Fatalf("CallTversion: want nil, got %v", err)
	}
	t.Logf("CallTversion: msize %v version %v", m, v)

	t.Logf("Server is %v", n.String())

	a, err := c.CallTattach(0, rpc.NOFID, "/", "")
	if err != nil {
		t.Fatalf("CallTattach: want nil, got %v", err)
	}

	t.Logf("Attach is %v", a)
	w, err := c.CallTwalk(0, 1, []string{"hi", "there"})
	if err == nil {
		t.Fatalf("CallTwalk(0,1,[\"hi\", \"there\"]): want err, got QIDS %v", w)
	}
	t.Logf("Walk is %v", w)
	w, err = c.CallTwalk(0, 1, []string{"etc", "hosts"})
	if err != nil {
		t.Fatalf("CallTwalk(0,1,[\"etc\", \"hosts\"]): want nil, got %v", err)
	}
	t.Logf("Walk is %v", w)

	of, _, err := c.CallTopen(22, rpc.OREAD)
	if err == nil {
		t.Fatalf("CallTopen(22, rpc.OREAD): want err, got nil")
	}
	of, _, err = c.CallTopen(1, rpc.OWRITE)
	if err == nil {
		t.Fatalf("CallTopen(0, rpc.OWRITE): want err, got nil")
	}
	of, _, err = c.CallTopen(1, rpc.OREAD)
	if err != nil {
		t.Fatalf("CallTopen(0, rpc.OREAD): want nil, got %v", nil)
	}
	t.Logf("Open is %v", of)

	b, err := c.CallTread(22, 0, 0)
	if err == nil {
		t.Fatalf("CallTread(22, 0, 0): want err, got nil")
	}
	b, err = c.CallTread(1, 1, 22)
	if err != nil {
		t.Fatalf("CallTread(0, 22, 1): want nil, got %v", err)
	}
	t.Logf("read is %v", string(b))

	// TODO: close the file, then try to read it.

}
