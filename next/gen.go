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
	"os"
	"reflect"
	"html/template"

	"github.com/rminnich/ninep/next"
)

const (
)

type emitter struct {
	MFunc string
	// Encoders always return []byte
	MParms *bytes.Buffer
	MList  *bytes.Buffer
	MCode  *bytes.Buffer

	// Decoders always take []byte as parms.
	UFunc string
	UList    *bytes.Buffer
	UCode    *bytes.Buffer
	URet     *bytes.Buffer
	inBWrite bool
}

type call struct {
	T *emitter
	R *emitter
}

type pack struct {
	t  interface{}
	tn string
	r  interface{}
	rn string
}

var (
	debug = nodebug //log.Printf
	packages = []*pack{
//		{t: next.RerrorPkt{}, tn: "Rerror", r: next.RerrorPkt{}, rn: "Rerror"},
		{t: next.TversionPkt{}, tn: "TversionPkt", r: next.RversionPkt{}, rn: "RversionPkt"},
//		{t: next.TattachPkt{}, tn: "Tattach", r: next.RattachPkt{}, rn: "Rattach"},
//		{t: next.TwalkPkt{}, tn: "Twalk", r: next.RwalkPkt{}, rn: "Rwalk"},
	}
	mtfunc = template.Must(template.New("mt").Parse(`func Marshal{{.MFunc}} (b *bytes.Buffer, t Tag{{.MParms}}) {
b.Reset()
tb.Write([]byte{0,0,0,0,
uint8({{.MFunc}})),
byte(t), byte(t>>8),
{{.MCode}}
{ l := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n")}
return
}
`))
	mrfunc = template.Must(template.New("mr").Parse(`func Unmarshal{{.UFunc}} (b *bytes.Buffer) ({{.URet}}, t Tag, err error) {
u uint8[8]
{{.UCode}}
return
}
`))
)

func nodebug(string, ...interface{}) {
}

func newCall() *call {
	c := &call{}
	c.T = &emitter{"", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, false}
	c.R = &emitter{"", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, false}
	return c
}

func emitEncodeInt(n string, l int, e *emitter) {
	debug("emit %v, %v", n, l)
	for i:= 0; i < l; i++ {
		if !e.inBWrite {
			e.MCode.WriteString("\tb.Write([]byte{")
			e.inBWrite = true
		}
		e.MCode.WriteString(fmt.Sprintf("\tuint8(%v>>%v),\n", n, i*8))
	}
}

func emitDecodeInt(n string, l int, e *emitter) {
	debug("emit %v, %v", n, l)
	e.UCode.WriteString(fmt.Sprintf("\tif _, err = b.Read(u[:%v]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint%v: need %v, have %%d\", b.Len())\n\treturn\n\t}\n", l, l*8, l))
	e.UCode.WriteString(fmt.Sprintf("\t%v = uint%d(u[0])\n", n, l*8))
	for i:= 1; i < l; i++ {
		e.UCode.WriteString(fmt.Sprintf("\t%v |= uint%d(u[%d]<<%v)\n", n, l*8, i, i*8))
	}
}

// TODO: templates.
func emitEncodeString(n string, e *emitter) {
	e.MCode.WriteString(fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", n, n))
	e.MCode.WriteString("\t})\n")
	e.inBWrite = false
	e.MCode.WriteString(fmt.Sprintf("\tb.Write([]byte(%v))\n", n))
}

// TODO: templates.
func emitDecodeString(n string, e *emitter) {
	emitDecodeInt("l", 2, e)
	e.UCode.WriteString(fmt.Sprintf("\tif b.Len() < l {\n\t\terr = fmt.Errorf(\"pkt too short for string: need %%d, have %%d\", l, b.Len())\n\treturn\n\t}\n"))
	e.UCode.WriteString(fmt.Sprintf("\t%v = b.String()\n}\n", n))
}

func genEncodeStruct(v interface{}, n string, e *emitter) error {
	debug("genEncodeStruct(%T, %v, %v)", v, n, e)
	t := reflect.ValueOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fn := t.Type().Field(i).Name
		debug("genEncodeStruct %T n %v field %d %v %v\n", t, n, i, f.Type(), f.Type().Name())
		genEncodeData(f.Interface(), n + "." + fn, e)
	}
	return nil
}

func genDecodeStruct(v interface{}, n string, e *emitter) error {
	debug("genDecodeStruct(%T, %v, %v)", v, n, e)
	t := reflect.ValueOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fn := t.Type().Field(i).Name
		debug("genDecodeStruct %T n %v field %d %v %v\n", t, n, i, f.Type(), f.Type().Name())
		genDecodeData(f.Interface(), n + "." + fn, e)
	}
	return nil
}

func genEncodeData(v interface{}, n string, e *emitter) error {
	debug("genEncodeData(%T, %v, %v)", v, n, e)
	s := reflect.ValueOf(v).Kind() 
	switch s {
	case reflect.Uint8:
		emitEncodeInt(n, 1, e)
	case reflect.Uint16:
		emitEncodeInt(n, 2, e)
	case reflect.Uint32:
		emitEncodeInt(n, 4, e)
	case reflect.Uint64:
		emitEncodeInt(n, 8, e)
	case reflect.String:
		emitEncodeString(n, e)
	case reflect.Struct:
		return genEncodeStruct(v, n, e)
		default:
			log.Printf("Can't handle type %v", s)
	}
	return nil
}

func genDecodeData(v interface{}, n string, e *emitter) error {
	debug("genEncodeData(%T, %v, %v)", v, n, e)
	s := reflect.ValueOf(v).Kind() 
	switch s {
	case reflect.Uint8:
		emitDecodeInt(n, 1, e)
	case reflect.Uint16:
		emitDecodeInt(n, 2, e)
	case reflect.Uint32:
		emitDecodeInt(n, 4, e)
	case reflect.Uint64:
		emitDecodeInt(n, 8, e)
	case reflect.String:
		emitDecodeString(n, e)
	case reflect.Struct:
		return genDecodeStruct(v, n, e)
		default:
			log.Printf("Can't handle type %v", s)
	}
	return nil
}

// genParms writes the parameters for declarations (name and type)
// a list of names (for calling the encoder)
func genParms(v interface{}, n string, e *emitter) error {
	comma := ", "
	t := reflect.ValueOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fn := t.Type().Field(i).Name
		e.MList.WriteString(comma + fn)
		e.MParms.WriteString(comma + fn + " " + f.Type().Name())
		comma = ", "
	}
	return nil
}

// genRets writes the rets for declarations (name and type)
// a list of names
func genRets(v interface{}, n string, e *emitter) error {
	comma := ""
	t := reflect.ValueOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fn := t.Type().Field(i).Name
		e.UList.WriteString(comma + fn)
		e.URet.WriteString(comma + fn + " " + f.Type().Name())
		comma = ", "
	}
	e.UList.WriteString(comma + "error")
	e.URet.WriteString(comma + "err error")

	return nil
}

// genMsgRPC generates the call and reply declarations and marshalers. We don't think of encoders as too separate
// because the 9p encoding is so simple.
func genMsgRPC(p *pack) (*call, error) {

	c := newCall()
	c.T.MFunc = p.tn
	c.R.UFunc = p.rn
	if err := genEncodeData(p.t, p.tn, c.T); err != nil {
		log.Fatalf("%v", err)
	}
	if err := genEncodeData(p.r, p.rn, c.R); err != nil {
		log.Fatalf("%v", err)
	}
	if err := genDecodeData(p.t, p.tn, c.T); err != nil {
		log.Fatalf("%v", err)
	}
	if err := genDecodeData(p.r, p.rn, c.R); err != nil {
		log.Fatalf("%v", err)
	}

	if err := genParms(p.t, p.tn, c.T); err != nil {
		log.Fatalf("%v", err)
	}

	if err := genRets(p.r, p.rn, c.R); err != nil {
		log.Fatalf("%v", err)
	}

	//log.Print("e %v d %v", c.T, c.R)

//	log.Print("------------------", c.T.MParms, "0", c.T.MList, "1", c.R.URet, "2", c.R.UList)
//	log.Print("------------------", c.T.MCode)
	mtfunc.Execute(os.Stdout, c.T)
	mrfunc.Execute(os.Stdout, c.R)

	return nil, nil

}

func main() {
	for _, p := range packages {
		_, err := genMsgRPC(p)
		if err != nil {
			log.Fatalf("%v", err)
		}
		//log.Printf("on return, call is %v", call)
	}

}
