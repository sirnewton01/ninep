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
	"fmt"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/rminnich/ninep/next"
)

var (
	packages = []struct {
		c  interface{}
		cn string
		r  interface{}
		rn string
	}{
		{c: next.RerrorPkt{}, cn: "Rerror", r: next.RerrorPkt{}, rn: "Rerror"},
		{c: next.TversionPkt{}, cn: "Tversion", r: next.RversionPkt{}, rn: "Rversion"},
	}
)

// For a given message type, gen generates declarations, return values, lists of variables, and code.
func gen(v interface{}, msg, prefix string) (eParms, eCode, eList, dRet, dCode, dList string, err error) {
	comma := ""
	var inBWrite bool = true
	//packet := msg + "Pkt"
	mvars := ""
	//	code := "\tvar u32 [4]byte\n\tvar u16 [2]byte\n\tvar l int\n"
	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	eCode = "\tb.Reset()\n\tb.Write([]byte{0,0,0,0, uint8(" + msg + "),\n\tbyte(t), byte(t>>8),\n"
	dCode = "\tvar u16 [2]byte\n\tvar l int\n"
	// Unmarshal will always return the tag in addition to everything else.
	dRet = ""

	t := reflect.TypeOf(v)
	dCode += "\tif _, err = b.Read(u16[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for tag: need 2, have %d\", b.Len())\n\treturn\n\t}\n"
	dCode += fmt.Sprintf("\tt = Tag(uint16(u16[0])|uint16(u16[1])<<8)\n")
	for i := 0; i < t.NumField(); i++ {
		if !inBWrite {
			eCode += "\tb.Write([]byte{"
			inBWrite = true
		}
		f := t.Field(i)
		Mn := "M" + prefix + f.Name
		Un := "U" + prefix + f.Name
		eParms += fmt.Sprintf(", %v %v", Mn, f.Type.Kind())
		eList += comma + Mn
		dRet += fmt.Sprintf("%v%v %v", comma, Un, f.Type.Kind())
		dList += comma + Un
		mvars += fmt.Sprintf("b, %v", Mn)
		switch f.Type.Kind() {
		case reflect.Uint32:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", Mn, Mn)
			eCode += fmt.Sprintf("uint8(%v>>16),uint8(%v>>24),\n", Mn, Mn)
			dCode += "\t{\n\tvar u32 [4]byte\n\tif _, err = b.Read(u32[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for uint32: need 4, have %d\", b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\t%v = uint32(u32[0])<<0|uint32(u32[1])<<8|uint32(u32[2])<<16|uint32(u32[3])<<24\n}\n", Un)
		case reflect.Uint16:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),\n", Mn, Mn)
			dCode += "\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\t%v = uint16(u16[0])|uint16(u16[1]<<8)\n", Un)
		case reflect.String:
			eCode += fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", Mn, Mn)
			if inBWrite {
				eCode += "\t})\n"
				inBWrite = false
			}
			eCode += fmt.Sprintf("\tb.Write([]byte(%v))\n", Mn)
			dCode += "\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\tl = int(u16[0])|int(u16[1]<<8)\n")
			dCode += "\tif b.Len() < l  {\n\t\terr = fmt.Errorf(\"pkt too short for string: need %d, have %d\", l, b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\t%v = b.String()\n", Un)
		default:
			err = fmt.Errorf("Can't encode %T.%v", v, f)
			return
		}
		comma = ", "
	}
	eCode += "\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n"

	return
}

// genMsgCoder tries to generate an encoder and a decoder and caller for a given message pair.
func genMsgRPC(tv interface{}, tmsg string, rv interface{}, rmsg string) (enc, dec, call, reply, dispatch string, err error) {
	tpacket := tmsg + "Pkt"
	eTParms, eTCode, eTList, dTRet, dTCode, _, err := gen(tv, tmsg, tmsg[0:1])
	if err != nil {
		return
	}

	rpacket := rmsg + "Pkt"
	eRParms, eRCode, _, dRRet, dRCode, dRList, err := gen(rv, rmsg, rmsg[0:1])
	if err != nil {
		return
	}
	enc = fmt.Sprintf("func Marshal%v (b *bytes.Buffer, t Tag%v) {\n%v\n\treturn\n}\n", tpacket, eTParms, eTCode)
	dec = fmt.Sprintf("func Unmarshal%v (b *bytes.Buffer) (%v, t Tag, err error) {\n%v\n\treturn\n}\n", tpacket, dTRet, dTCode)
	if rmsg == "Rerror" {
		return
	}
	dec += fmt.Sprintf("func Unmarshal%v (b *bytes.Buffer) (%v, t Tag, err error) {\n%v\n\treturn\n}\n", rpacket, dRRet, dRCode)
	enc += fmt.Sprintf("func Marshal%v (b *bytes.Buffer, t Tag%v) {\n%v\n\treturn\n}\n", rpacket, eRParms, eRCode)
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
}`, tmsg, tmsg, eTList, dRList, dRList, dRList, rmsg, dRList)

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
}`, rmsg, eTList[2:], tmsg, dRList, rmsg, eTList[2:], rmsg, dRList)

	call = fmt.Sprintf("func (c *Client)Call%v (%v) (%v, err error) {\n%v\n /*%v / %v */\n", tmsg, eTParms[2:], dRRet, callCode, eTList, dRList)

	dispatch = fmt.Sprintf("case %v:\n\ts.Trace(\"%v\")\n\ts.Srv%v(b)\n", tmsg, tmsg, rmsg)

	return enc + "\n//=====================\n",
		dec + "\n//=====================\n",
		/*mvars  + */ call + "\n//=====================\n",
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
	if err := ioutil.WriteFile("genout.go", []byte(out), 0600); err != nil {
		log.Fatalf("%v", err)
	}
}
