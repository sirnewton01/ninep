// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ufs

import (
	"flag"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/rminnich/ninep/rpc"
	"github.com/rminnich/ninep/next"
)

type File struct {
	rpc.QID
	fullName string
	File *os.File
}

type FileServer struct {
	mu sync.Mutex
	root *File
	Versioned bool
	Files map[rpc.FID] *File
	IOunit rpc.MaxSize
}

var (
	debug = flag.Int("debug", 0, "print debug messages")
	root = flag.String("root", "/", "Set the root for all attaches")
)
func (e FileServer) Rversion(msize rpc.MaxSize, version string) (rpc.MaxSize, string, error) {
	if version != "9P2000" {
		return 0, "", fmt.Errorf("%v not supported; only 9P2000", version)
	}
	e.Versioned = true
	return msize, version, nil
}

func (e FileServer) Rattach(fid rpc.FID, afid rpc.FID, aname string, _ string) (rpc.QID, error) {
	if afid != rpc.NOFID {
		return rpc.QID{}, fmt.Errorf("We don't do auth attach")
	}
	// There should be no .. or other such junk in the Aname. Clean it up anyway.
fmt.Fprintf(os.Stderr, "-------------------------- %v ---------------", []byte(aname))
	aname = path.Join("/", aname)
	aname = path.Join(*root, aname)
fmt.Fprintf(os.Stderr, "=================== stat %v =====================", aname)
	_, _ = os.Stat("FUCK")
	st, err := os.Stat(aname)
fmt.Fprintf(os.Stderr, "=================== %v %v =====================", st, err)
	if err != nil {
		return rpc.QID{}, err
	}
	r := &File{fullName: aname,}
	r.QID = dir2QID(st)
	e.Files[fid] = r
	e.root = r
	return r.QID, nil
}

func (e FileServer) Rflush(f rpc.FID, t rpc.FID) error {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	return fmt.Errorf("Read: bad rpc.FID %v", f)
}

func (e FileServer) Rwalk(fid rpc.FID, newfid rpc.FID, paths []string) ([]rpc.QID, error) {
	e.mu.Lock()
	f, ok := e.Files[fid]
	e.mu.Unlock()
	if ! ok {
		return nil, fmt.Errorf("Bad FID")
	}
	if len(paths) == 0 {
		e.mu.Lock()
		defer e.mu.Unlock()
		of, ok := e.Files[newfid]
		if ok {
			return nil, fmt.Errorf("FID in use")
		}
		e.Files[newfid] = of
		return []rpc.QID{of.QID, }, nil
	}
	p := f.fullName
	q := make([]rpc.QID, len(paths))

	// optional: path.Join(p, paths[i]...) and see if the endpoint exists and return if not.
	// It can save a little time. Maybe later.
	for i := range paths {
		p = path.Join(p, paths[i])
		st, err := os.Lstat(p)
		if err != nil {
			return nil, fmt.Errorf("ENOENT")
		}
		q[i] = dir2QID(st)
	}
	st, err := os.Lstat(p)
	if err != nil {
		return nil, fmt.Errorf("Walk succeeded but stat failed")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	// this is quite unlikely, which is why we don't bother checking for it first.
	if _, ok := e.Files[newfid]; ok {
		return nil, fmt.Errorf("FID in use")
	}
	e.Files[newfid] = &File{fullName: p, QID: dir2QID(st)}
	return q, nil
}
	


func (e FileServer) Ropen(fid rpc.FID, mode rpc.Mode) (rpc.QID, rpc.MaxSize, error) {
	e.mu.Lock()
	f, ok := e.Files[fid]
	e.mu.Unlock()
	if ! ok {
		return rpc.QID{}, 0, fmt.Errorf("Bad FID")
	}

	var err error
	f.File, err = os.OpenFile(f.fullName, omode2uflags(mode), 0)
	if err != nil {
		return rpc.QID{}, 0, err
	}

	return f.QID, e.IOunit, nil
}
func (e FileServer) Rcreate(fid rpc.FID, name string, perm rpc.Perm, mode rpc.Mode) (rpc.QID, rpc.MaxSize, error) {
	//fmt.Printf("open(%v, %v\n", fid, mode)
	return rpc.QID{}, 5000, nil
}
func (e FileServer) Rclunk(f rpc.FID) error {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	//fmt.Printf("clunk(%v)\n", f)
	return fmt.Errorf("Clunk: bad rpc.FID %v", f)
}
func (e FileServer) Rstat(f rpc.FID) (rpc.Dir, error) {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return rpc.Dir{}, nil
	}
	//fmt.Printf("stat(%v)\n", f)
	return rpc.Dir{}, fmt.Errorf("Stat: bad rpc.FID %v", f)
}
func (e FileServer) Rwstat(f rpc.FID, d rpc.Dir) error {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	//fmt.Printf("stat(%v)\n", f)
	return fmt.Errorf("Wstat: bad rpc.FID %v", f)
}
func (e FileServer) Rremove(f rpc.FID) error {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	//fmt.Printf("remove(%v)\n", f)
	return fmt.Errorf("Remove: bad rpc.FID %v", f)
}
func (e FileServer) Rread(f rpc.FID, o rpc.Offset, c rpc.Count) ([]byte, error) {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return []byte("HI"), nil
	}
	return nil, fmt.Errorf("Read: bad rpc.FID %v", f)
}

func (e FileServer) Rwrite(f rpc.FID, o rpc.Offset, c rpc.Count, b []byte) (rpc.Count, error) {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return c, nil
	}
	return -1, fmt.Errorf("Write: bad rpc.FID %v", f)
}

type ServerOpt func(*rpc.Server) error

func NewUFS(opts ...rpc.ServerOpt) (*rpc.Server, error) {
	f := FileServer{}
	f.Files = make(map[rpc.FID] *File)
	// any opts for the ufs layer can be added here too ...
	s, err := next.NewServer(f, opts...)
	if err != nil {
		return nil, err
	}
	f.IOunit = 8192
	s.Start()
	return s, nil
}

