// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package next

import (
	"flag"
	"fmt"

	rpc "github.com/rminnich/ninep/rpc"
)

var debug = flag.Int("debug", 0, "print debug messages")

func (e *FileServer) Rversion(msize rpc.MaxSize, version string) (rpc.MaxSize, string, error) {
	if version != "9P2000" {
		return 0, "", fmt.Errorf("%v not supported; only 9P2000", version)
	}
	e.versioned = true
	return msize, version, nil
}

func (e *FileServer) Rattach(rpc.FID, rpc.FID, string, string) (rpc.QID, error) {
	if !e.versioned {
		return rpc.QID{}, fmt.Errorf("Attach: Version must be done first")
	}
	return rpc.QID{}, nil
}

func (e *FileServer) Rflush(f rpc.FID, t rpc.FID) error {
	if !e.versioned {
		return fmt.Errorf("Attach: Version must be done first")
	}
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	return fmt.Errorf("Read: bad rpc.FID %v", f)
}

func (e *FileServer) Rwalk(fid rpc.FID, newfid rpc.FID, paths []string) ([]rpc.QID, error) {
	//fmt.Printf("walk(%d, %d, %d, %v\n", fid, newfid, len(paths), paths)
	if len(paths) > 1 {
		return nil, fmt.Errorf("%v: No such file or directory", paths)
	}
	switch paths[0] {
	case "null":
		return []rpc.QID{rpc.QID{Type: 0, Version: 0, Path: 0xaa55}}, nil
	}
	return nil, fmt.Errorf("%v: No such file or directory", paths)
}

func (e *FileServer) Ropen(fid rpc.FID, mode rpc.Mode) (rpc.QID, rpc.MaxSize, error) {
	//fmt.Printf("open(%v, %v\n", fid, mode)
	return rpc.QID{}, 4000, nil
}
func (e *FileServer) Rcreate(fid rpc.FID, name string, perm rpc.Perm, mode rpc.Mode) (rpc.QID, rpc.MaxSize, error) {
	//fmt.Printf("open(%v, %v\n", fid, mode)
	return rpc.QID{}, 5000, nil
}
func (e *FileServer) Rclunk(f rpc.FID) error {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	//fmt.Printf("clunk(%v)\n", f)
	return fmt.Errorf("Clunk: bad rpc.FID %v", f)
}
func (e *FileServer) Rstat(f rpc.FID) (rpc.Dir, error) {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return rpc.Dir{}, nil
	}
	//fmt.Printf("stat(%v)\n", f)
	return rpc.Dir{}, fmt.Errorf("Stat: bad rpc.FID %v", f)
}
func (e *FileServer) Rwstat(f rpc.FID, d rpc.Dir) error {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	//fmt.Printf("stat(%v)\n", f)
	return fmt.Errorf("Wstat: bad rpc.FID %v", f)
}
func (e *FileServer) Rremove(f rpc.FID) error {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return nil
	}
	//fmt.Printf("remove(%v)\n", f)
	return fmt.Errorf("Remove: bad rpc.FID %v", f)
}
func (e *FileServer) Rread(f rpc.FID, o rpc.Offset, c rpc.Count) ([]byte, error) {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return []byte("HI"), nil
	}
	return nil, fmt.Errorf("Read: bad rpc.FID %v", f)
}

func (e *FileServer) Rwrite(f rpc.FID, o rpc.Offset, c rpc.Count, b []byte) (rpc.Count, error) {
	switch int(f) {
	case 2:
		// Make it fancier, later.
		return c, nil
	}
	return -1, fmt.Errorf("Write: bad rpc.FID %v", f)
}
