// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package next

import (
	rpc "github.com/rminnich/ninep/rpc"
)

// A File is defined by a QID. File Servers never see a FID.
// There are two channels. The first is for normal requests.
// The second is for Flushes. File server code always
// checks the flush channel first. At the same time, server code
// puts the flush into both channels, so the server code has some
// idea when the flush entered the queue. This is very similar
// to how MSI-X works on PCIe ...
type File struct {
}

// Server maintains file system server state. This is inclusive of RPC
// server state plus more. In our case when we walk to a fid we kick
// off a goroutine to manage it. As a result we need a map of Tag to FID
// so we know what to do about Tflush.
type FileServer struct {
	rpc.Server
	versioned bool
	// Files we have walked to. For each FID, there is a goroutine
	// serving that File.
	Files map[rpc.FID] File
	// Active operations. For each Tag we've been asked to work on,
	// we enter the File in the scoreboard.
	ScoreBoard map[rpc.Tag]File
}
