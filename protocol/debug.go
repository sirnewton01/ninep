// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocol

import (
	"bytes"
        "flag"
	"log"
)

var (
        Debug = flag.Bool("debug", false, "print debug messages")
)

type DebugServer struct {
	NineServer
}

func (e *DebugServer) Rversion(msize MaxSize, version string) (MaxSize, string, error) {
	log.Printf(">>> Tversion %v %v\n", msize, version)
	msize, version, err := e.NineServer.Rversion(msize, version)
	if err == nil {
		log.Printf("<<< Rversion %v %v\n", msize, version)
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return msize, version, err
}

func (e *DebugServer) Rattach(fid FID, afid FID, uname string, aname string) (QID, error) {
	log.Printf(">>> Tattach fid %v,  afid %v, uname %v, aname %v\n", fid, afid,
		uname, aname)
	qid, err := e.NineServer.Rattach(fid, afid, uname, aname)
	if err == nil {
		log.Printf("<<< Rattach %v\n", qid)
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return qid, err
}

func (e *DebugServer) Rflush(o Tag) error {
	log.Printf(">>> Tflush tag %v\n", o)
	err := e.NineServer.Rflush(o)
	if err == nil {
		log.Printf("<<< Rflush\n")
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return err
}

func (e *DebugServer) Rwalk(fid FID, newfid FID, paths []string) ([]QID, error) {
	log.Printf(">>> Twalk fid %v, newfid %v, paths %v\n", fid, newfid, paths)
	qid, err := e.NineServer.Rwalk(fid, newfid, paths)
	if err == nil {
		log.Printf("<<< Rwalk %v\n", qid)
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return qid, err
}

func (e *DebugServer) Ropen(fid FID, mode Mode) (QID, MaxSize, error) {
	log.Printf(">>> Topen fid %v, mode %v\n", fid, mode)
	qid, iounit, err := e.NineServer.Ropen(fid, mode)
	if err == nil {
		log.Printf("<<< Ropen %v %v\n", qid, iounit)
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return qid, iounit, err
}

func (e *DebugServer) Rcreate(fid FID, name string, perm Perm, mode Mode) (QID, MaxSize, error) {
	log.Printf(">>> Tcreate fid %v, name %v, perm %v, mode %v\n", fid, name,
		perm, mode)
	qid, iounit, err := e.NineServer.Rcreate(fid, name, perm, mode)
	if err == nil {
		log.Printf("<<< Rcreate %v %v\n", qid, iounit)
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return qid, iounit, err
}

func (e *DebugServer) Rclunk(fid FID) error {
	log.Printf(">>> Tclunk fid %v\n", fid)
	err := e.NineServer.Rclunk(fid)
	if err == nil {
		log.Printf("<<< Rclunk\n")
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return err
}

func (e *DebugServer) Rstat(fid FID) ([]byte, error) {
	log.Printf(">>> Tstat fid %v\n", fid)
	b, err := e.NineServer.Rstat(fid)
	if err == nil {
		dir, _ := Unmarshaldir(bytes.NewBuffer(b))
		log.Printf("<<< Rstat %v\n", dir)
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return b, err
}

func (e *DebugServer) Rwstat(fid FID, b []byte) error {
	dir, _ := Unmarshaldir(bytes.NewBuffer(b))
	log.Printf(">>> Twstat fid %v, %v\n", fid, dir)
	err := e.NineServer.Rwstat(fid, b)
	if err == nil {
		log.Printf("<<< Rwstat\n")
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return err
}

func (e *DebugServer) Rremove(fid FID) error {
	log.Printf(">>> Tremove fid %v\n", fid)
	err := e.NineServer.Rremove(fid)
	if err == nil {
		log.Printf("<<< Rremove\n")
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return err
}

func (e *DebugServer) Rread(fid FID, o Offset, c Count) ([]byte, error) {
	log.Printf(">>> Tread fid %v, off %v, count %v\n", fid, o, c)
	b, err := e.NineServer.Rread(fid, o, c)
	if err == nil {
		log.Printf("<<< Rread %v\n", len(b))
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return b, err
}

func (e *DebugServer) Rwrite(fid FID, o Offset, b []byte) (Count, error) {
	log.Printf(">>> Twrite fid %v, off %v, count %v\n", fid, o, len(b))
	c, err := e.NineServer.Rwrite(fid, o, b)
	if err == nil {
		log.Printf("<<< Rwrite %v\n", c)
	} else {
		log.Printf("<<< Error %v\n", err)
	}
	return c, err
}
