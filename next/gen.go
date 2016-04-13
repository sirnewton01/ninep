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
		{c: next.TwalkPkt{}, cn: "Twalk", r: next.RwalkPkt{}, rn: "Rwalk"},
	}
)

// For a given message type, gen generates declarations, return values, lists of variables, and code.
func decl(em *emitter, v interface{}, msg, prefix string) {
	y := reflect.ValueOf(v)
	t := y.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		k := f.Type.Kind()
		Mn := "M" + prefix + f.Name
		Un := "U" + prefix + f.Name
			em.MList.WriteString(em.comma + Mn)
			em.UList.WriteString(em.comma + Un)

			switch k {
			case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.String:
				em.MParms.WriteString(fmt.Sprintf(", %v %v", Mn, f.Type.Kind()))
				em.URet.WriteString(fmt.Sprintf("%v%v %v", em.comma, Un, f.Type.Kind()))
			default:
				em.MParms.WriteString(fmt.Sprintf(", %v %v", Mn, f.Name))
				em.URet.WriteString(fmt.Sprintf("%v%v %v", em.comma, Un, f.Name))
			}
			em.comma = ", "
	}
}

func emitNum(em *emitter, l int, Mn, Un string) {
	em.UCode.WriteString(fmt.Sprintf("\tif _, err = b.Read(u[:%v]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint%v: need %v, have %%d\", b.Len())\n\treturn\n\t}\n", l, l*8, l))
	for i:= 0; i < l; i++ {
		if !em.inBWrite {
			em.MCode.WriteString("\tb.Write([]byte{")
			em.inBWrite = true
		}
		// TODO: do straight inline for as many bytes as are needed. This is inefficient.
		em.MCode.WriteString(fmt.Sprintf("\tuint8(%v>>%v),\n", Mn, i*8))
		em.UCode.WriteString(fmt.Sprintf("\t%v |= uint%d(u[%d]<<%v)\n", Un, l*8, i, i*8))
	}
}
func emitString(em *emitter, Mn, Un string) {
		if !em.inBWrite {
			em.MCode.WriteString("\tb.Write([]byte{")
			em.inBWrite = true
		}
			em.MCode.WriteString(fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", Mn, Mn))
			em.MCode.WriteString("\t})\n")
			em.inBWrite = false
			em.MCode.WriteString(fmt.Sprintf("\tb.Write([]byte(%v))\n", Mn))
			em.UCode.WriteString("\tif _, err = b.Read(u[:2]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
			em.UCode.WriteString(fmt.Sprintf("\t{ var l = int(u[0])|int(u[1]<<8)\n"))
			em.UCode.WriteString("\tif b.Len() < l  {\n\t\terr = fmt.Errorf(\"pkt too short for string: need %d, have %d\", l, b.Len())\n\treturn\n\t}\n")
			em.UCode.WriteString(fmt.Sprintf("\t%v = b.String()\n}\n", Un))
}

func emitSlice(em *emitter, Mn, Un string) {
			em.MCode.WriteString(fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", Mn, Mn))
			em.MCode.WriteString(fmt.Sprintf("\tfor _,v := range %v {\n", Mn))
			em.MCode.WriteString("\t}\n")
			
			em.UCode.WriteString("\tif _, err = b.Read(u[:2]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
			em.UCode.WriteString(fmt.Sprintf("\t%v = uint16(u[0])|uint16(u[1]<<8)\n", Un))

}

func emitStringSlice(em *emitter, Mn, Un string) {
		if !em.inBWrite {
			em.MCode.WriteString("\tb.Write([]byte{")
			em.inBWrite = true
		}
			em.MCode.WriteString(fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),})\n", Mn, Mn))
			em.inBWrite = false
			em.MCode.WriteString(fmt.Sprintf("\tfor _,v := range %v {\n", Mn))
			emitString(em, "v", Un)
			em.MCode.WriteString("\n\t}\n")
			
			em.UCode.WriteString("\tif _, err = b.Read(u[:2]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
			em.UCode.WriteString(fmt.Sprintf("\t%v = uint16(u[0])|uint16(u[1]<<8)\n", Un))

}
// For a given message type, gen generates declarations, return values, lists of variables, and code.
// TODO: this is crap. I'm still learning.
func genStruct(em *emitter, v interface{}, msg, prefix string) error {
	y := reflect.ValueOf(v)
	t := y.Type()
	for i := 0; i < t.NumField(); i++ {
		if !em.inBWrite {
			em.MCode.WriteString("\tb.Write([]byte{")
			em.inBWrite = true
		}
		f := t.Field(i)
		k := f.Type.Kind()
		Mn := "M" + prefix + f.Name
		Un := "U" + prefix + f.Name

		switch {
		case k == reflect.Uint64:
			emitNum(em, 8, Mn, Un)
		case k == reflect.Uint32:
			emitNum(em, 4, Mn, Un)
		case k == reflect.Uint16:
			emitNum(em, 2, Mn, Un)
		case k == reflect.Uint8:
			emitNum(em, 1, Mn, Un)
		case k == reflect.String:
			emitString(em, Mn, Un)
		case k == reflect.Struct:
			if err := genStruct(em, y.Field(i).Interface(), msg, prefix+f.Name+"."); err != nil {
				return err
			}
		// 9p encodes data length and wqid and arrays with different length lengths. Oh well.
		// TODO: clean this mess up.
		case k == reflect.Slice:
			fmt.Printf("SLICE!:")
			// encode. Unlike all other cases we have to generate an encoder for a variable array.
			switch f.Type.String() {
			// []byte in 9p uses int32 for length, unlike others.
			case "[]byte":
			em.MCode.WriteString(fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", Mn, Mn))
			em.MCode.WriteString(fmt.Sprintf("uint8(%v>>16),uint8(%v>>24),\n", Mn, Mn))
			em.UCode.WriteString("\t{\n\tvar u32 [4]byte\n\tif _, err = b.Read(u32[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for uint32: need 4, have %d\", b.Len())\n\treturn\n\t}\n")
			em.UCode.WriteString(fmt.Sprintf("\t%v = uint32(u32[0])<<0|uint32(u32[1])<<8|uint32(u32[2])<<16|uint32(u32[3])<<24\n}\n", Un))
			case "[]string":
				emitStringSlice(em, Mn, Un)
			default:
				emitSlice(em, Mn, Un)
			}
			// length set up, now it's a loop.
			// This can really be done MUCH BETTER ...
			fmt.Printf("'%s' '%v\n'", f.Type.String(), y.Field(i).Interface())
		case k == reflect.Array:
			fmt.Printf("array? really?\n", f.Name)
		default:
			return fmt.Errorf("Can't encode k is '%v', '%v' f.Type '%v', f.Name %v", k, f, f.Type, f.Name)
		}
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
	// Why generate i here? So we have a handy variable big enough to hold everything.
	// Why use it for the tag? So we ensure it's used at least once. Doing this saves a bunch of
	// foolishness.
	em.UCode.WriteString("\tvar u [8]byte\n\tvar i uint64\n\t")
	em.inBWrite = true
	// Unmarshal will always return the tag in addition to everything else.
	em.UCode.WriteString("\tif _, err = b.Read(u[:2]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for tag: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
	em.UCode.WriteString(fmt.Sprintf("\ti = uint64(uint16(u[0])|uint16(u[1])<<8)\n\tt = Tag(i)\n"))
	decl(em, tv, tmsg, tmsg[0:1])
	err = genStruct(em, tv, tmsg, tmsg[0:1])
	if err != nil {
		return
	}
	em.MCode.WriteString("\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n")

	rpacket := rmsg + "Pkt"
	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	dm.MCode.WriteString("\tb.Reset()\n\tb.Write([]byte{0,0,0,0, uint8(" + rmsg + "),\n\tbyte(t), byte(t>>8),\n")
	dm.UCode.WriteString("\tvar u [8]byte\n\t")
	dm.inBWrite = true
	// Unmarshal will always return the tag in addition to everything else.
	dm.UCode.WriteString("\tif _, err = b.Read(u[:2]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for tag: need 2, have %d\", b.Len())\n\treturn\n\t}\n")
	dm.UCode.WriteString(fmt.Sprintf("\tt = Tag(uint16(u[0])|uint16(u[1])<<8)\n"))
	decl(dm, rv, rmsg, rmsg[0:1])
	if err = genStruct(dm, rv, rmsg, rmsg[0:1]); err != nil {
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
	var u [8]byte
	// This can't really happen. 
	if _, err := b.Read(u[:2]); err != nil {
		return
	}
	t := Tag(uint16(u[0])|uint16(u[1])<<8)
	MarshalRerrorPkt (b, t, s)
}
`
	if err := ioutil.WriteFile("genout.go", []byte(out), 0600); err != nil {
		log.Fatalf("%v", err)
	}
}
