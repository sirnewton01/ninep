// Copyright 2015 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore
//
package main

import (
	"fmt"
	"log"
//	"next9p"
	"reflect"
)

// genMsgCoder tries to generate an encoder and a decoder for a given message type.
func genMsgRPC(v interface{}) (string, string, error) {
	var e, d string
	var inBWrite bool
	n := fmt.Sprintf("%T", v)
	p := n[5:]
	n = n[5:len(n)-3]
	e = fmt.Sprintf("func Marshal%v(b bytes.Buffer", n)
	d = fmt.Sprintf("func Unmarshall%v([]byte) (", n)
	eParms := ""
	dRet := p + ", error) {\n"
	eCode := ""
	dCode := ""

	// Add the encoding boiler plate: 4 bytes of size to be filled in later,
	// The tag type, and the tag itself.
	eCode += "\tb.Write([]byte{0,0,0,0})\n\tb.Write([]byte{uint8("+n+"),\n"
	inBWrite = true

	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		if ! inBWrite {
			eCode += "\tb.Write([]byte{"
			inBWrite = true
		}
		f := t.Field(i)
		eParms += ", "
		n := f.Name
		eParms += fmt.Sprintf("%v %v", n, f.Type.Kind())
		switch f.Type.Kind() {
		case reflect.Uint32:
			eCode += fmt.Sprintf("\tuint8(%v>>24),uint8(%v>>16),", n, n)
			fallthrough
		case reflect.Uint16:
			eCode += fmt.Sprintf("\tuint8(%v>>8),uint8(%v),\n", n, n)
		case reflect.String:
			eCode += fmt.Sprintf("\tuint8(len(%v)>>8),uint8(len(%v)),\n", n, n)
			if inBWrite {
				eCode += "\t})\n"
				inBWrite = false
			}
			eCode += fmt.Sprintf("\tb.Write([]byte(%v))\n", n)
		default:
			return "", "", fmt.Errorf("Can't encode %T.%v", v, f)
		}

	}
	if inBWrite {
		eCode += "\t})\n"
	}
	eCode += "\tl := b.Len()\n\tcopy(b.Bytes(), []byte{uint8(l>>24), uint8(l>>16), uint8(l>>8), uint8(l)})\n"
	return e+eParms+") {\n"+eCode+"}\n", d+ dRet+dCode+"\n}\n" , nil
}

func main() {
	e, d, err := genMsgRPC(TversionPkt{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Printf("package main\n\nimport \"bytes\"\n%v \n %v \n", e, d)
}
