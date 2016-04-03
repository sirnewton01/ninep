// Copyright 2015 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"
	"github.com/rminnich/ninep/next"
	"io/ioutil"
	"log"
	"reflect"
)

var (
	packages = []struct {
		p interface{}
		n string
	}{
		{p: next.TversionPkt{}, n: "Tversion"},
		{p: next.RversionPkt{}, n: "Rversion"},
	}
)

// For a given message type, gen generates declarations, return values, lists of variables, and code.
func gen(v interface{}, msg string) (eParms, eCode, dRet, dCode string, err error) {
	comma := ""
	var inBWrite bool = true
	//packet := msg + "Pkt"
	mvars := ""
	//	code := "\tvar u32 [4]byte\n\tvar u16 [2]byte\n\tvar l int\n"
	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	eCode = "\tb.Write([]byte{0,0,0,0, uint8(" + msg + "), 0, 0,\n"
	dCode = "\tvar u32 [4]byte\n\tvar u16 [2]byte\n\tvar l int\n"

	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		if !inBWrite {
			eCode += "\tb.Write([]byte{"
			inBWrite = true
		}
		f := t.Field(i)
		n := f.Name
		eParms += fmt.Sprintf(", %v %v", n, f.Type.Kind())
		dRet += fmt.Sprintf("%v%v %v", comma, n, f.Type.Kind())
		comma = ", "
		mvars += fmt.Sprintf("b, %v", n)
		switch f.Type.Kind() {
		case reflect.Uint32:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),", n, n)
			eCode += fmt.Sprintf("\tuint8(%v>>16),uint8(%v>>24),\n", n, n)
			dCode += "\tif _, err = b.Read(u32[:]); err != nil {\n\terr = fmt.Errorf(\"pkt too short for uint32: need 4, have %d\", b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\t%v = uint32(u32[0])<<24|uint32(u32[1])<<16|uint32(u32[2])<<8|uint32(u32[3])\n", n)
		case reflect.Uint16:
			eCode += fmt.Sprintf("\tuint8(%v),uint8(%v>>8),\n", n, n)
			dCode += "\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\t%v = uint16(u16[0])<<8|uint16(u16[1])\n", n)
		case reflect.String:
			eCode += fmt.Sprintf("\tuint8(len(%v)),uint8(len(%v)>>8),\n", n, n)
			if inBWrite {
				eCode += "\t})\n"
				inBWrite = false
			}
			eCode += fmt.Sprintf("\tb.Write([]byte(%v))\n", n)
			dCode += "\tif _, err = b.Read(u16[:]); err != nil {\n\t\terr = fmt.Errorf(\"pkt too short for uint16: need 2, have %d\", b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\tl = int(u16[0])<<8|int(u16[1])\n")
			dCode += "\tif b.Len() < l  {\n\t\terr = fmt.Errorf(\"pkt too short for string: need %d, have %d\", l, b.Len())\n\treturn\n\t}\n"
			dCode += fmt.Sprintf("\t%v = b.String()\n", n)
		default:
			err = fmt.Errorf("Can't encode %T.%v", v, f)
			return
		}

	}
	eCode += "\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l), uint8(l>>8), uint8(l>>16), uint8(l>>24)})\n"
	return
}

// genMsgCoder tries to generate an encoder and a decoder for a given message type.
func genMsgRPC(v interface{}, msg string) (e, d, call string, err error) {
	packet := msg + "Pkt"
	eParms, eCode, dRet, dCode, err := gen(v, msg)
	if err != nil {
		return
	}

	enc := fmt.Sprintf("func Marshal%v (b *bytes.Buffer%v) {\n%v\n\treturn\n}\n", packet, eParms, eCode)
	dec := fmt.Sprintf("func Unmarshal%v (b *bytes.Buffer) (%v, err error) {\n%v\n\treturn\n}\n", packet, dRet, dCode)
	return enc + "\n//=====================\n",
		dec + "\n//=====================\n",
		/*mvars  + */ call + "\n//=====================\n", nil
}

func main() {
	var enc, dec, call string
	for _, p := range packages {
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
