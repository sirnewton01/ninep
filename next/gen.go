// Copyright 2015 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"github.com/rminnich/ninep/next"
	"reflect"
)


var (
	packages = []struct {
		p interface{}
		n string
		}  {
		{p: next.TversionPkt{}, n: "Tversion"},
		{p: next.RversionPkt{}, n: "Rversion"},
	}
)

// genMsgCoder tries to generate an encoder and a decoder for a given message type.
/*
func genMsgRPC(v interface{}, packet, msg string) (e, d, call string, err error) {
	var inBWrite bool
	//packageType := fmt.Sprintf("%T", v)
	e = fmt.Sprintf("func Marshal%v(b *bytes.Buffer", packet)
	d = fmt.Sprintf("func Unmarshall%v(d[]byte) (*", packet)
	call = fmt.Sprintf("func (c *Client)Call%v(", packet)
	eParms := ""
	dRet := packet + fmt.Sprintf(", error) {\n\tvar p *%v\n\tb := bytes.NewBuffer(d)\n", packet)
	eCode := ""
	dCode := "\tvar u32 [4]byte\n\tvar u16 [2]byte\n\tvar l int\n"

	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	inBWrite = true
	eCode += "\tb.Write([]byte{0,0,0,0, uint8(" + msg + "), uint8(0), uint8(0>>8),\n"

	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		if !inBWrite {
			eCode += "\tb.Write([]byte{"
			inBWrite = true
		}
		f := t.Field(i)
		n := f.Name
		parms := fmt.Sprintf(", %v %v", n, f.Type.Kind())
		eParms += Parms
		callParms += Parms
callcode starts with b := bytes.Buffer Marshall%v (packet) ( then add parms in loop
callcode is then )\n\tr := make(chan []byte)\n\tc.FromClient <- &RPC{b.Bytes(), r)\nb = <-r
unmarshall(b into the type)
return the type you want.
		switch f.Type.Kind() {
		case reflect.Uint32:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", n, n)
			eCode += fmt.Sprintf("\tuint8(%v>>16),uint8(%v>>24),\n", n, n)
			dCode += "\tif _, err := b.Read(u32[:]); err != nil {\n\t\treturn nil, fmt.Errorf(\"pkt too short for uint32: need 4, have %d\", b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\tp.%v = uint32(u32[0])<<24|uint32(u32[1])<<16|uint32(u32[2])<<8|uint32(u32[3])\n", n)
		case reflect.Uint16:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),\n", n, n)
			dCode += "\tif _, err := b.Read(u16[:]); err != nil {\n\t\treturn nil, fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\tp.%v = uint16(u16[0])<<8|uint16(u16[1])\n", n)
		case reflect.String:
			eCode += fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", n, n)
			if inBWrite {
				eCode += "\t})\n"
				inBWrite = false
			}
			eCode += fmt.Sprintf("\tb.Write([]byte(%v))\n", n)
			dCode += "\tif _, err := b.Read(u16[:]); err != nil {\n\t\treturn nil, fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\tl = int(u16[0])<<8|int(u16[1])\n")
			dCode += "\tif b.Len() < l  {\n\t\treturn nil, fmt.Errorf(\"pkt too short for string: need %d, have %d\", l, b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\tp.%v = b.String()\n", n)
		default:
			return "", "", fmt.Errorf("Can't encode %T.%v", v, f)
		}

	}
	if inBWrite {
		eCode += "\t})\n"
	}
	eCode += "\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n"
	return e + eParms + ") {\n" + eCode + "}\n", d + dRet + dCode + "\n\treturn p, nil\n}\n", nil
}
 */
// genMsgCoder tries to generate an encoder and a decoder for a given message type.
func genMsgRPC(v interface{}, msg string) (e, d, call string, err error) {
	var inBWrite bool = true
	packet := msg + "Pkt"
	decoderParms := "b []byte"
	var vars string
	mvars := "b"
	code := "\tvar u32 [4]byte\n\tvar u16 [2]byte\n\tvar l int\n"
	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	eCode := "\tb.Write([]byte{0,0,0,0, uint8(" + msg + "), 0, 0,\n"
	dCode := "\tvar u32 [4]byte\n\tvar u16 [2]byte\n\tvar l int\n"

	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		if !inBWrite {
			eCode += "\tb.Write([]byte{"
			inBWrite = true
		}
		f := t.Field(i)
		n := f.Name
		vars += fmt.Sprintf(", %v %v", n, f.Type.Kind())
		mvars += fmt.Sprintf(", %v", n)
		switch f.Type.Kind() {
		case reflect.Uint32:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", n, n)
			eCode += fmt.Sprintf("\tuint8(%v>>16),uint8(%v>>24),\n", n, n)
			dCode += "\tif _, err := b.Read(u32[:]); err != nil {\n\t\treturn nil, fmt.Errorf(\"pkt too short for uint32: need 4, have %d\", b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\t%v := uint32(u32[0])<<24|uint32(u32[1])<<16|uint32(u32[2])<<8|uint32(u32[3])\n", n)
		case reflect.Uint16:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),\n", n, n)
			dCode += "\tif _, err := b.Read(u16[:]); err != nil {\n\t\treturn nil, fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\t%v = uint16(u16[0])<<8|uint16(u16[1])\n", n)
		case reflect.String:
			eCode += fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", n, n)
			if inBWrite {
				eCode += "\t})\n"
				inBWrite = false
			}
			eCode += fmt.Sprintf("\tb.Write([]byte(%v))\n", n)
			dCode += "\tif _, err := b.Read(u16[:]); err != nil {\n\t\treturn nil, fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\tl = int(u16[0])<<8|int(u16[1])\n")
			dCode += "\tif b.Len() < l  {\n\t\treturn nil, fmt.Errorf(\"pkt too short for string: need %d, have %d\", l, b.Len())\n\t}\n"
			dCode += fmt.Sprintf("\tp.%v = b.String()\n", n)
		default:
			return "", "", "", fmt.Errorf("Can't encode %T.%v", v, f)
		}

	}

	enc := fmt.Sprintf("func Marshall%v (b *bytes.Buffer%v) {\n%v\n\treturn}\n", packet, vars, eCode)
/*
	dRet := packet + fmt.Sprintf(", error) {\n\tvar p *%v\n\tb := bytes.NewBuffer(d)\n", packet)
	if inBWrite {
		eCode += "\t})\n"
	}
	eCode += "\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n"
	return e + eParms + ") {\n" + eCode + "}\n", d + dRet + dCode + "\n\treturn p, nil\n}\n", nil
 */
	return enc + "\n=====================\n" , 
	decoderParms + dCode  + "\n=====================\n" , 
	mvars  + code  + "\n=====================\n" , nil
}

func main() {
	var enc, dec, call string
	for _, p := range packages  {
		e, d, c, err := genMsgRPC(p.p, p.n)
		if err != nil {
			log.Fatalf("%v", err)
		}
		enc += e
		dec += d
		call += c
	}
	out := "package next\n\nimport (\n\t\"bytes\"\n\t\"fmt\"\n)\n" + enc + "\n" + dec + "\n\n" + call
	if err := ioutil.WriteFile("genout.go", []byte(out), 0600); err != nil {
		log.Fatalf("%v", err)
	}
}
