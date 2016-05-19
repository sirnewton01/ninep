package ufs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
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

// a simple prototype file system.
type makeit struct {
	n string      // name
	m os.FileMode // mode
	s string      // for symlinks or content
}

var tests = []makeit{
	{
		n: "ro",
		m: 0444,
		s: "",
	},
	{
		n: "rw",
		m: 0666,
		s: "",
	},
	{
		n: "wo",
		m: 0222,
		s: "",
	},
}

func TestMount(t *testing.T) {
	/* Create the simple file system. */
	tmpdir, err := ioutil.TempDir(os.TempDir(), "hi.dir")
	if err != nil {
		t.Fatalf("%v", err)
	}

	for i := range tests {
		if err := ioutil.WriteFile(path.Join(tmpdir, tests[i].n), []byte("hi"), tests[i].m); err != nil {
			t.Fatalf("%v", err)
		}
	}

	sr, cw := io.Pipe()
	cr, sw := io.Pipe()

	c, err := rpc.NewClient(func(c *rpc.Client) error {
		c.FromNet, c.ToNet = cr, cw
		return nil
	},
		func(c *rpc.Client) error {
			c.Msize = 8192
			c.Trace = print //t.Logf
			return nil
		})
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("Client is %v", c.String())

	n, err := NewUFS(func(s *rpc.Server) error {
		s.FromNet, s.ToNet = sr, sw
		s.Trace = print //t.Logf
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
	ro := strings.Split(path.Join(tmpdir, "ro"), "/")

	w, err = c.CallTwalk(0, 1, ro)
	if err != nil {
		t.Fatalf("CallTwalk(0,1,%v): want nil, got %v", ro, err)
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
		t.Fatalf("CallTopen(1, rpc.OREAD): want nil, got %v", nil)
	}
	t.Logf("Open is %v", of)

	b, err := c.CallTread(22, 0, 0)
	if err == nil {
		t.Fatalf("CallTread(22, 0, 0): want err, got nil")
	}
	b, err = c.CallTread(1, 1, 1)
	if err != nil {
		t.Fatalf("CallTread(1, 1, 1): want nil, got %v", err)
	}
	t.Logf("read is %v", string(b))

	/* make sure Twrite fails */
	if _, err = c.CallTwrite(1, 0, b); err == nil {
		t.Fatalf("CallTwrite(1, 0, b): want err, got nil")
	}

	d, err := c.CallTstat(1)
	if err != nil {
		t.Fatalf("CallTstat(1): want nil, got %v", err)
	}
	t.Logf("stat is %v", d)

	d, err = c.CallTstat(22)
	if err == nil {
		t.Fatalf("CallTstat(22): want err, got nil)")
	}
	t.Logf("stat is %v", d)

	if err := c.CallTclunk(22); err == nil {
		t.Fatalf("CallTclunk(22): want err, got nil")
	}
	if err := c.CallTclunk(1); err != nil {
		t.Fatalf("CallTclunk(1): want nil, got %v", err)
	}
	if _, err := c.CallTread(1, 1, 22); err == nil {
		t.Fatalf("CallTread(1, 1, 22) after clunk: want err, got nil")
	}

	d, err = c.CallTstat(1)
	if err == nil {
		t.Fatalf("CallTstat(1): after clunk: want err, got nil")
	}
	t.Logf("stat is %v", d)

	// fun with write
	rw := strings.Split(path.Join(tmpdir, "rw"), "/")
	w, err = c.CallTwalk(0, 1, rw)
	if err != nil {
		t.Fatalf("CallTwalk(0,1,%v): want nil, got %v", rw, err)
	}
	t.Logf("Walk is %v", w)

	of, _, err = c.CallTopen(1, rpc.OREAD)
	if err != nil {
		t.Fatalf("CallTopen(1, rpc.OREAD): want nil, got %v", nil)
	}
	if err := c.CallTclunk(1); err != nil {
		t.Fatalf("CallTclunk(1): want nil, got %v", err)
	}
	w, err = c.CallTwalk(0, 1, rw)
	if err != nil {
		t.Fatalf("CallTwalk(0,1,%v): want nil, got %v", rw, err)
	}
	t.Logf("Walk is %v", w)

	of, _, err = c.CallTopen(1, rpc.OWRITE)
	if err != nil {
		t.Fatalf("CallTopen(0, rpc.OWRITE): want nil, got %v", err)
	}
	t.Logf("open OWRITE of is %v", of)
	if _, err = c.CallTwrite(1, 1, []byte("there")); err != nil {
		t.Fatalf("CallTwrite(1, 0, \"there\"): want nil, got %v", err)
	}
	if _, err = c.CallTwrite(22, 1, []byte("there")); err == nil {
		t.Fatalf("CallTwrite(22, 1, \"there\"): want err, got nil")
	}

	// readdir test.
	w, err = c.CallTwalk(0, 3, []string{})
	if err != nil {
		t.Fatalf("CallTwalk(0,3,[]string{}): want nil, got %v", err)
	}
	t.Logf("Walk is %v", w)
	w, err = c.CallTwalk(0, 2, strings.Split(tmpdir, "/"))
	if err != nil {
		t.Fatalf("CallTwalk(0,2,strings.Split(tmpdir, \"/\")): want nil, got %v", err)
	}
	t.Logf("Walk is %v", w)
	of, _, err = c.CallTopen(2, rpc.OWRITE)
	if err == nil {
		t.Fatalf("CallTopen(2, rpc.OWRITE) on /: want err, got nil")
	}
	of, _, err = c.CallTopen(2, rpc.ORDWR)
	if err == nil {
		t.Fatalf("CallTopen(2, rpc.ORDWR) on /: want err, got nil")
	}
	of, _, err = c.CallTopen(2, rpc.OREAD)
	if err != nil {
		t.Fatalf("CallTopen(1, rpc.OREAD): want nil, got %v", nil)
	}
	var o rpc.Offset
	var iter int
	for iter < 10 {
		iter++
		b, err = c.CallTread(2, o, 256)
		if fmt.Sprintf("%v", err) == "EOF" {
			break
		}
		if err != nil {
			t.Fatalf("CallTread(2, 0, 256): want nil, got %v", err)
		}

		dent, err := rpc.Unmarshaldir(bytes.NewBuffer(b))
		if err != nil {
			t.Errorf("Unmarshalldir: want nil, got %v", err)
		}
		t.Logf("dir read is %v", dent)
		o += rpc.Offset(len(b))
	}

	if iter > 9 {
		t.Errorf("Too many reads from the directory: want 3, got %v", iter)
	}
	if err := c.CallTclunk(2); err != nil {
		t.Fatalf("CallTclunk(1): want nil, got %v", err)
	}
}
