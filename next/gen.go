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
	"io/ioutil"
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

var (
	packages = []struct {
		c  interface{}
		cn string
		r  interface{}
		rn string
	}{
		{c: next.RerrorPkt{}, cn: "Rerror", r: next.RerrorPkt{}, rn: "Rerror"},
		{c: next.TversionPkt{}, cn: "Tversion", r: next.RversionPkt{}, rn: "Rversion"},
		{c: next.TattachPkt{}, cn: "Tattach", r: next.RattachPkt{}, rn: "Rattach"},
//		{c: next.TwalkPkt{}, cn: "Twalk", r: next.RwalkPkt{}, rn: "Rwalk"},
	}
)

func code(em *emitter, k reflect.Kind, f reflect.StructField, Mn, Un string) error {
	var err error
	switch {
	case k == reflect.Uint64:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v>>16),uint8(%v>>24),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v)>>32,uint8(%v>>40),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v>>48),uint8(%v>>56),\n", Mn, Mn))
		em.UCode.WriteString("\t{\n\tvar u64 [8]byte\n\tif _, err = b.Read(u64[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for uint64: need 8, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v |= uint64(u64[0])<<0|uint64(u64[1])<<8|uint64(u64[2])<<16|uint64(u64[3])<<24\n", Un))
		em.UCode.WriteString(fmt.Sprintf("\t%v |= uint64(u64[0])<<32|uint64(u64[1])<<40|uint64(u64[2])<<48|uint64(u64[3])<<56\n}\n", Un))
	case k == reflect.Uint32:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v>>16),uint8(%v>>24),\n", Mn, Mn))
		em.UCode.WriteString("\t{\n\tvar u32 [4]byte\n\tif _, err = b.Read(u32[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for uint32: need 4, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v = uint32(u32[0])<<0|uint32(u32[1])<<8|uint32(u32[2])<<16|uint32(u32[3])<<24\n}\n", Un))
	case k == reflect.Uint16:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),uint8(%v>>8),\n", Mn, Mn))
		em.UCode.WriteString("\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v = uint16(u16[0])|uint16(u16[1]<<8)\n", Un))
	case k == reflect.String:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", Mn, Mn))
		if em.inBWrite {
			em.MCode.WriteString("\t})\n")
			em.inBWrite = false
		}
		em.MCode.WriteString(fmt.Sprintf("\tb.Write([]byte(%v))\n", Mn))
		em.UCode.WriteString("\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t{ var l = int(u16[0])|int(u16[1]<<8)\n"))
		em.UCode.WriteString("\tif b.Len() < l  {\n\t\terr = fmt.Errorf(\"pkt too short for string: need %d, have %d\", l, b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v = b.String()\n}\n", Un))

		// one option is to call ourselves with this but let's try this for now.
		// There are very few types in 9p.
	case f.Name == "QID":
		em.MParms.WriteString(fmt.Sprintf(", %v QID", Mn))
		em.URet.WriteString(fmt.Sprintf("%v%v QID", em.comma, Un))
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v.Type),", Mn))

		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v.Version),uint8(%v.Version>>8),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Version>>16),uint8(%v.Version>>24),\n", Mn, Mn))

		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>0),uint8(%v.Path>>8),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>16),uint8(%v.Path>>24),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>32),uint8(%v.Path>>40),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>48),uint8(%v.Path>>56),\n", Mn, Mn))

		em.UCode.WriteString("\tif b.Len() < QIDLen {\n\t\terr = fmt.Errorf(\"pkt too short for QID: need 13, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString("{ q := b.Bytes()\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v.Type = q[0]\n", Un))
		for i := 0; i < 4; i++ {
			em.UCode.WriteString(fmt.Sprintf("\t%v.Version = uint32(q[%d+1])<<%d\n", Un, i, i*8))
		}
		for i := 0; i < 8; i++ {
			em.UCode.WriteString(fmt.Sprintf("\t%v.Path |= uint64(q[%d+5])<<%d\n", Un, i, i*8))
		}
		em.UCode.WriteString("\n}")
	default:
		err = fmt.Errorf("Can't encode %v f.Type %v", f, f.Type)
		return err
	}
	return err
}

// For a given message type, gen generates declarations, return values, lists of variables, and code.
func gen(em *emitter, v interface{}, msg, prefix string) error {
	t := reflect.TypeOf(v)
	y := reflect.ValueOf(v)
	for i := 0; i < t.NumField(); i++ {
		if !em.inBWrite {
			em.MCode.WriteString("\tb.Write([]byte{")
			em.inBWrite = true
		}
		f := t.Field(i)
		Mn := "M" + prefix + f.Name
		Un := "U" + prefix + f.Name
		em.MList.WriteString(em.comma + Mn)
		em.UList.WriteString(em.comma + Un)

		k := f.Type.Kind()
		switch k {
		case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.String:
			em.MParms.WriteString(fmt.Sprintf(", %v %v", Mn, f.Type.Kind()))
			em.URet.WriteString(fmt.Sprintf("%v%v %v", em.comma, Un, f.Type.Kind()))
		}

		//
		//if err := code(em, k, f, Mn, Un); err != nil {
		//			return err
		//}
	switch {
	case k == reflect.Uint64:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v>>16),uint8(%v>>24),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v)>>32,uint8(%v>>40),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v>>48),uint8(%v>>56),\n", Mn, Mn))
		em.UCode.WriteString("\t{\n\tvar u64 [8]byte\n\tif _, err = b.Read(u64[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for uint64: need 8, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v |= uint64(u64[0])<<0|uint64(u64[1])<<8|uint64(u64[2])<<16|uint64(u64[3])<<24\n", Un))
		em.UCode.WriteString(fmt.Sprintf("\t%v |= uint64(u64[0])<<32|uint64(u64[1])<<40|uint64(u64[2])<<48|uint64(u64[3])<<56\n}\n", Un))
	case k == reflect.Uint32:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v>>16),uint8(%v>>24),\n", Mn, Mn))
		em.UCode.WriteString("\t{\n\tvar u32 [4]byte\n\tif _, err = b.Read(u32[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for uint32: need 4, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v = uint32(u32[0])<<0|uint32(u32[1])<<8|uint32(u32[2])<<16|uint32(u32[3])<<24\n}\n", Un))
	case k == reflect.Uint16:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),uint8(%v>>8),\n", Mn, Mn))
		em.UCode.WriteString("\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v = uint16(u16[0])|uint16(u16[1]<<8)\n", Un))
	case k == reflect.Uint8:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),\n", Mn))
		em.UCode.WriteString("\tif _, err = b.Read(u16[:1]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint8: need 1, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v = uint8(u16[0])\n", Un))
	case k == reflect.String:
		em.MCode.WriteString(fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", Mn, Mn))
		if em.inBWrite {
			em.MCode.WriteString("\t})\n")
			em.inBWrite = false
		}
		em.MCode.WriteString(fmt.Sprintf("\tb.Write([]byte(%v))\n", Mn))
		em.UCode.WriteString("\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t{ var l = int(u16[0])|int(u16[1]<<8)\n"))
		em.UCode.WriteString("\tif b.Len() < l  {\n\t\terr = fmt.Errorf(\"pkt too short for string: need %d, have %d\", l, b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v = b.String()\n}\n", Un))

	case f.Name == "QID":
		// TODO: make this work recursively.
		if false {
			if err := gen(em, y.Field(i).Interface(), msg + "."+f.Name, prefix); err != nil {
				return err
			}
		}
		em.MParms.WriteString(fmt.Sprintf(", %v QID", Mn))
		em.URet.WriteString(fmt.Sprintf("%v%v QID", em.comma, Un))
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v.Type),", Mn))

		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v.Version),uint8(%v.Version>>8),", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Version>>16),uint8(%v.Version>>24),\n", Mn, Mn))

		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>0),uint8(%v.Path>>8),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>16),uint8(%v.Path>>24),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>32),uint8(%v.Path>>40),\n", Mn, Mn))
		em.MCode.WriteString(fmt.Sprintf("uint8(%v.Path>>48),uint8(%v.Path>>56),\n", Mn, Mn))

		em.UCode.WriteString("\tif b.Len() < QIDLen {\n\t\terr = fmt.Errorf(\"pkt too short for QID: need 13, have %d\", b.Len())\n\treturn\n\t}\n")
		em.UCode.WriteString("{ q := b.Bytes()\n")
		em.UCode.WriteString(fmt.Sprintf("\t%v.Type = q[0]\n", Un))
		for i := 0; i < 4; i++ {
			em.UCode.WriteString(fmt.Sprintf("\t%v.Version = uint32(q[%d+1])<<%d\n", Un, i, i*8))
		}
		for i := 0; i < 8; i++ {
			em.UCode.WriteString(fmt.Sprintf("\t%v.Path |= uint64(q[%d+5])<<%d\n", Un, i, i*8))
		}
		em.UCode.WriteString("\n}")
	default:
		return fmt.Errorf("Can't encode %v f.Type %v", f, f.Type)
	}

		em.comma = ", "
	}
	if em.inBWrite {
		em.MCode.WriteString("\t})\n")
		em.inBWrite = false
	}
	return nil
}

// genMsgCoder tries to generate an encoder and a decoder and caller for a given message pair.
func genMsgRPC(tv interface{}, tmsg string, rv interface{}, rmsg string) (enc, dec, call, reply, dispatch string, err error) {
	em := &emitter{&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", false}
	dm := &emitter{&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", false}
	tpacket := tmsg + "Pkt"
	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	em.MCode.WriteString("\tb.Reset()\n\tb.Write([]byte{0,0,0,0, uint8(" + tmsg + "),\n\tbyte(t), byte(t>>8),\n")
	em.UCode.WriteString("\tvar u16 [2]byte\n\t")
	em.inBWrite = true
	// Unmarshal will always return the tag in addition to everything else.
	em.UCode.WriteString("\tif _, err = b.Read(u16[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for tag: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
	em.UCode.WriteString(fmt.Sprintf("\tt = Tag(uint16(u16[0])|uint16(u16[1])<<8)\n"))
	err = gen(em, tv, tmsg, tmsg[0:1])
	if err != nil {
		return
	}
	em.MCode.WriteString("\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n")

	rpacket := rmsg + "Pkt"
	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	dm.MCode.WriteString("\tb.Reset()\n\tb.Write([]byte{0,0,0,0, uint8(" + rmsg + "),\n\tbyte(t), byte(t>>8),\n")
	dm.UCode.WriteString("\tvar u16 [2]byte\n\t")
	dm.inBWrite = true
	// Unmarshal will always return the tag in addition to everything else.
	dm.UCode.WriteString("\tif _, err = b.Read(u16[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for tag: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
	dm.UCode.WriteString(fmt.Sprintf("\tt = Tag(uint16(u16[0])|uint16(u16[1])<<8)\n"))
	if err = gen(dm, rv, rmsg, rmsg[0:1]); err != nil {
		return
	}

	dm.MCode.WriteString("\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n")


fmt.Printf("EM :%v:\nDM:%v:\n", em, dm)
	enc = fmt.Sprintf("func Marshal%v (b *bytes.Buffer, t Tag%v) {\n%v\n\treturn\n}\n", tpacket, em.MParms, em.MCode)
	dec = fmt.Sprintf("func Unmarshal%v (b *bytes.Buffer) (%v, t Tag, err error) {\n%v\n\treturn\n}\n", tpacket, em.URet, em.UCode)
	if rmsg == "Rerror" {
		return
	}
	dec += fmt.Sprintf("func Unmarshal%v (b *bytes.Buffer) (%v, t Tag, err error) {\n%v\n\treturn\n}\n", rpacket, dm.URet, dm.UCode)
	enc += fmt.Sprintf("func Marshal%v (b *bytes.Buffer, t Tag%v) {\n%v\n\treturn\n}\n", rpacket, dm.MParms, dm.MCode)
	// The call code takes teh same paramaters as encode, and has the parameters of decode.
	// We use named parameters so that on stuff like Read we can return the []b we were passed.
	// I guess that's stupid, since we *could* just not return the []b, but OTOH this is more
	// consistent?

	// TODO: use templates. Please. really soon. this sucks.
	callCode := fmt.Sprintf(`var b = bytes.Buffer{}
c.Trace("%v")
t := <- c.Tags
r := make (chan []byte)
c.Trace(":tag %%v, FID %%v", t, c.FID)
Marshal%vPkt(&b, t, %v)
c.FromClient <- &RPCCall{b: b.Bytes(), Reply: r}
bb := <-r
if MType(bb[4]) == Rerror {
	s, _, err := UnmarshalRerrorPkt(bytes.NewBuffer(bb[5:]))
	if err != nil {
		return %v, err
	}
	return %v, fmt.Errorf("%%v", s)
} else {
	%v, _, err = Unmarshal%vPkt(bytes.NewBuffer(bb[5:]))
}
return %v, err
}`, tmsg, tmsg, em.MList, dm.UList, dm.UList, dm.UList, rmsg, dm.UList)

	reply = fmt.Sprintf(`func (s *Server) Srv%v(b*bytes.Buffer) (err error) {
	%v, t, err := Unmarshal%vPkt(b)
	//if err != nil {
	//}
	%v, err := s.NS.%v(%v)
if err != nil {
	MarshalRerrorPkt(b, t, fmt.Sprintf("%%v", err))
} else {
	Marshal%vPkt(b, t, %v)
}
	return nil
}
`, rmsg, string(em.MList.Bytes()[2:]), tmsg, dm.UList, rmsg, string(em.MList.Bytes()[2:]), rmsg, dm.UList)

	call = fmt.Sprintf("func (c *Client)Call%v (%v) (%v, err error) {\n%v\n /*%v / %v */\n", tmsg, string(em.MParms.Bytes()[2:]), dm.URet, callCode, em.MList, dm.UList)

	dispatch = fmt.Sprintf("case %v:\n\ts.Trace(\"%v\")\n\ts.Srv%v(b)\n", tmsg, tmsg, rmsg)

	return enc + "\n//=====================\n",
		dec + "\n//=====================\n",
		call + "\n//=====================\n",
		reply, dispatch, nil

}

func main() {
	var enc, dec, call, reply string
	dispatch := `func dispatch(s *Server, b *bytes.Buffer) error {
t := MType(b.Bytes()[4])
switch(t) {
`
	for _, p := range packages {
		e, d, c, r, s, err := genMsgRPC(p.c, p.cn, p.r, p.rn)
		if err != nil {
			log.Fatalf("%v", err)
		}
		// We do Rerror first and ignore 3 of 5 things.
		enc += e
		dec += d
		call += c
		reply += r
		dispatch += s
	}
	dispatch += "\n\tdefault: log.Fatalf(\"Can't happen: bad packet type 0x%x\\n\", t)\n}\nreturn nil\n}"
	out := "package next\n\nimport (\n\t\"bytes\"\n\t\"fmt\"\n\t\"log\"\n)\n" + enc + "\n" + dec + "\n\n" + call + "\n\n" + reply + "\n\n" + dispatch
	out += `
func ServerError (b *bytes.Buffer, s string) {
	var u16 [2]byte
	// This can't really happen. 
	if _, err := b.Read(u16[:]); err != nil {
		return
	}
	t := Tag(uint16(u16[0])|uint16(u16[1])<<8)
	MarshalRerrorPkt (b, t, s)
}
`
	if err := ioutil.WriteFile("genout.go", []byte(out), 0600); err != nil {
		log.Fatalf("%v", err)
	}
}
