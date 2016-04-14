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
	"fmt"
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
//		{c: next.RerrorPkt{}, cn: "Rerror", r: next.RerrorPkt{}, rn: "Rerror"},
		{c: next.TversionPkt{}, cn: "TversionPkt", r: next.RversionPkt{}, rn: "RversionPkt"},
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

func emitInt(n string, l int, e *emitter) {
	log.Printf("emit %v, %v", n, l)
	for i:= 0; i < l; i++ {
		if !e.inBWrite {
			e.MCode.WriteString("\tb.Write([]byte{")
			e.inBWrite = true
		}
		e.MCode.WriteString(fmt.Sprintf("\tuint8(%v>>%v),\n", n, i*8))
	}
}

func genEncodeStruct(v interface{}, n string, e *emitter) error {
	log.Printf("genEncodeStruct(%T, %v, %v)", v, n, e)
	t := reflect.ValueOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fn := t.Type().Field(i).Name
		log.Printf("genEncodeStruct %T n %v field %d %v %v\n", t, n, i, f.Type(), f.Type().Name())
		genEncodeData(f.Interface(), n + "." + fn, e)
	}
	return nil
}

func genEncodeData(v interface{}, n string, e *emitter) error {
	log.Printf("genEncodeData(%T, %v, %v)", v, n, e)
	s := reflect.ValueOf(v).Kind() 
	switch s {
	case reflect.Uint16:
		emitInt(n, 2, e)
	case reflect.Struct:
		return genEncodeStruct(v, n, e)
		default:
			log.Printf("Can't handle type %v", s)
	}
	return nil
}
// genMsgRPC generates the call and reply declarations and marshalers. We don't think of encoders as too separate
// because the 9p encoding is so simple.
func genMsgRPC(p *pack) (*call, error) {
	c := newCall()
	err := genEncodeData(p.c, p.cn, c.T)
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
