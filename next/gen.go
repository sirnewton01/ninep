// Copyright 2015 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// gen is an rpc generator for the Plan 9 style XDR. It uses the types and structs
// defined in types. go. A core function, gen, creates the needed lists of
// parameters, code, and variable list for calling a Marshall function; and the
// return declaration, code, and return value list for an unmarshall function.
// You can think of an RPC as a pipline:
// marshal(parms) -> b[]byte over a network -> unmarshal -> dispatch -> reply(parms) -> unmarshal
// Since we have T messages and R messages in 9p, we adopt the following naming convention for, e.g., Version:
// MarshalTPktVersion
// UnmarshalTpktVersion
// MarshalRPktVersion
// UnmarshalRPktVersion
//
// A caller uses the MarshalT* and UnmarshallR* information. A dispatcher
// uses the  UnmarshalT* and MarshalR* information.
// Hence the caller needs the call MarshalT params, and UnmarshalR* returns;
// a dispatcher needs the UnmarshalT returns, and the MarshalR params.
package main

import (
	"bytes"
	"log"
	"reflect"

	"github.com/rminnich/ninep/next"
)

type emitter struct {
	// Encoders always return []byte
	MParms *bytes.Buffer
	MList  *bytes.Buffer
	MCode  *bytes.Buffer

	// Decoders always take []byte as parms.
	UList    *bytes.Buffer
	UCode    *bytes.Buffer
	URet     *bytes.Buffer
	comma    string
	inBWrite bool
}

type call struct {
	T *emitter
	R *emitter
}

type pack struct {
	c  interface{}
	cn string
	r  interface{}
	rn string
}

var (
	packages = []*pack{
		{c: next.RerrorPkt{}, cn: "Rerror", r: next.RerrorPkt{}, rn: "Rerror"},
//		{c: next.TversionPkt{}, cn: "Tversion", r: next.RversionPkt{}, rn: "Rversion"},
//		{c: next.TattachPkt{}, cn: "Tattach", r: next.RattachPkt{}, rn: "Rattach"},
//		{c: next.TwalkPkt{}, cn: "Twalk", r: next.RwalkPkt{}, rn: "Rwalk"},
	}
)

func newCall() *call {
	c := &call{}
	c.T = &emitter{&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", false}
	c.R = &emitter{&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", false}
	return c
}

func genStruct(v interface{}, e *emitter) error {
}

func genData(v interface{}, e *emitter) error {
	s := reflect.ValueOf(v).Kind() 
	switch s {
	case reflect.Struct:
		log.Printf("struct")
		default:
			log.Printf("Can't handle type %v", s)
	}
	return nil
}
// genMsgRPC generates the call and reply declarations and marshalers. We don't think of encoders as too separate
// because the 9p encoding is so simple.
func genMsgRPC(p *pack) (*call, error) {
	c := newCall()
	err := genData(p.c, c.T)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return nil, nil

}

func main() {
	for _, p := range packages {
		call, err := genMsgRPC(p)
		if err != nil {
			log.Fatalf("%v", err)
		}
		log.Printf("on return, call is %v", call)
	}

}
