// Copyright 2015 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore
//
package main

import (
	"fmt"
//	"next9p"
	"reflect"
)

// genMsgCoder tries to generate an encoder and a decoder for a given message type.
func genMsgRPC(v interface{}) (string, string, error) {
	var e, d string
	n := fmt.Sprintf("%T", v)
	n = n[6:]
	e = fmt.Sprintf("func Marshal%v(", n)
	d = fmt.Sprintf("func Unmarshall%v([]byte) (", n)
	eParms := ""
	dRet := ""
	
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if i > 0 {
			eParms += ", "
			dRet += ", "
		}
	fmt.Printf("%v %v\n", f.Name, f.Type.Kind)
		eParms += fmt.Sprintf("_%d %T", i, f)
		dRet += fmt.Sprintf("%T", f)
		switch f.Type.Kind() {
		case reflect.Uint16:
			fmt.Printf("uint16 ...\n")
		case reflect.Uint32:
			fmt.Printf("uint32 ...\n")
		case reflect.String:
			fmt.Printf("string ..\n")
		default:
			return "", "", fmt.Errorf("Can't encode %T.%v", v, f)
		}

	}
	return e+eParms, d+dRet, nil
}

func main() {
	e, d, err := genMsgRPC(TversionPkt{})
	fmt.Printf("%v \n %v \n %v\n ", e, d, err)
}
