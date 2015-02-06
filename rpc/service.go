// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"log"
	"reflect"
	"unicode"
	"unicode/utf8"
)

type ServiceMap map[string]*Service

type Service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

type methodType struct {
	method    reflect.Method // receiver method
	argsType  reflect.Type   // type of the request argument
	replyType reflect.Type   // type of the response argument
}

// Precompute the reflect type for error.  Can't use error directly
// because Typeof takes an empty interface value.  This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

func (s *Service) Call(req Request) (interface{}, error) {
	// Find Method
	serviceMethod := s.method[req.MethodName()]
	if serviceMethod == nil {
		ErrMethodNotFound.Data = req.MethodName()
		return nil, ErrMethodNotFound
	}

	var argv, replyv reflect.Value

	// Decode the argument value.
	argIsValue := false // if true, need to indirect before calling.
	if serviceMethod.argsType.Kind() == reflect.Ptr {
		argv = reflect.New(serviceMethod.argsType.Elem())
	} else {
 		argv = reflect.New(serviceMethod.argsType)
 		argIsValue = true
 	}

	// Decode the args.
	req.DecodeParams(argv.Interface())

	if argIsValue {
 		argv = argv.Elem()
 	}

	// Call the service method.
	replyv = reflect.New(serviceMethod.replyType.Elem())

	function := serviceMethod.method.Func

	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv,})

	errInter := returnValues[0].Interface()
	if err, ok := errInter.(error); ok && err != nil {
		return nil, err
	}

	return replyv.Interface(), nil
} 

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// installValidMethods returns valid Rpc methods of typ.
func installValidMethods(typ reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)

	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name

		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}

		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			log.Println("method", mname, "has wrong number of ins:", mtype.NumIn())
			continue
		}

		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			log.Println(mname, "argument type not exported:", argType)
			continue
		}

		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Ptr {
			log.Println("method", mname, "reply type not a pointer:", replyType)
			continue
		}

		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			log.Println("method", mname, "reply type not exported:", replyType)
			continue
		}

		// Method needs one out.
		if mtype.NumOut() != 1 {
			log.Println("method", mname, "has wrong number of outs:", mtype.NumOut())
			continue
		}

		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			log.Println("method", mname, "returns", returnType.String(), "not error")
			continue
		}

		methods[mname] = &methodType{
			method:    method, 
			argsType:  argType, 
			replyType: replyType,
		}
	}
	return methods
}