// Copyright 2012 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// This code is imported from the old ninep repo,
// with some changes.

package ufs

import (
	"os"
	"syscall"

	"github.com/Harvey-OS/ninep/stub"
)

func OModeToUnixFlags(mode stub.Mode) int {
	ret := int(0)
	switch mode & 3 {
	case stub.OREAD:
		ret = os.O_RDONLY
		break

	case stub.ORDWR:
		ret = os.O_RDWR
		break

	case stub.OWRITE:
		ret = os.O_WRONLY
		break

	case stub.OEXEC:
		ret = os.O_RDONLY
		break
	}

	if mode&stub.OTRUNC != 0 {
		ret |= os.O_TRUNC
	}

	return ret
}

// IsBlock reports if the file is a block device
func isBlock(d os.FileInfo) bool {
	sysif := d.Sys()
	if sysif == nil {
		return false
	}
	stat := sysif.(*syscall.Stat_t)
	return (stat.Mode & syscall.S_IFMT) == syscall.S_IFBLK
}

// IsChar reports if the file is a character device
func isChar(d os.FileInfo) bool {
	sysif := d.Sys()
	if sysif == nil {
		return false
	}
	stat := sysif.(*syscall.Stat_t)
	return (stat.Mode & syscall.S_IFMT) == syscall.S_IFCHR
}

func fileInfoToQID(d os.FileInfo) stub.QID {
	var qid stub.QID
	sysif := d.Sys()

	// on systems with inodes, use it.
	if sysif != nil {
		stat := sysif.(*syscall.Stat_t)
		qid.Path = uint64(stat.Ino)
	} else {
		qid.Path = uint64(d.ModTime().UnixNano())
	}

	qid.Version = uint32(d.ModTime().UnixNano() / 1000000)
	qid.Type = dirToQIDType(d)

	return qid
}

func dirToQIDType(d os.FileInfo) uint8 {
	ret := uint8(0)
	if d.IsDir() {
		ret |= stub.QTDIR
	}

	if d.Mode()&os.ModeSymlink != 0 {
		ret |= stub.QTSYMLINK
	}

	return ret
}

func dirTo9p2000Mode(d os.FileInfo) uint32 {
	ret := uint32(d.Mode() & 0777)
	if d.IsDir() {
		ret |= stub.DMDIR
	}
	return ret
}

func dirTo9p2000Dir(fi os.FileInfo) (*stub.Dir, error) {
	d := &stub.Dir{}
	d.QID = fileInfoToQID(fi)
	d.Mode = dirTo9p2000Mode(fi)
	// TODO: use info on systems that have it.
	d.Atime = uint32(fi.ModTime().Unix()) // uint32(atime(sysMode).Unix())
	d.Mtime = uint32(fi.ModTime().Unix())
	d.Length = uint64(fi.Size())
	d.Name = fi.Name()
	d.User = "root"
	d.Group = "root"

	return d, nil
}
