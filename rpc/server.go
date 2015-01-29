// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"log"
	"net/http"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
	"errors"
)

// Precompute the reflect type for error.  Can't use error directly
// because Typeof takes an empty interface value.  This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type service struct {
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

// Client request. When the client sends a request it is in
// the format "Service.Method"
type Request interface {
	ServiceName() string      
	MethodName()  string    
	DecodeParams(interface{}) error  
}

//-----------------------------------------------------------------------------
// Server
//-----------------------------------------------------------------------------

// Server represents an RPC Server.
type Server struct {
	mu         sync.RWMutex         // protects the serviceMap
	serviceMap map[string]*service
}

func NewServer() *Server {
	return &Server{
		serviceMap: make(map[string]*service),
	}
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//
//	- exported method
//	- two arguments, both of exported type
//	- the second argument is a pointer
//	- one return value, of type error
//
// It returns an error if the receiver is not an exported type or has
// no suitable methods.
//
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (server *Server) Register(rcvr interface{}) error {
	return server.RegisterName("", rcvr)
}

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func (server *Server) RegisterName(name string, rcvr interface{}) error {
	server.mu.Lock()
	defer server.mu.Unlock()

	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*service)
	}

	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()

	if !isExported(sname) {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}

	if name != "" {
		sname = name
	}

	if _, present := server.serviceMap[sname]; present {
		return errors.New("rpc: service already defined: " + sname)
	}

	s.name = sname
	s.method = installValidMethods(s.typ)

	if len(s.method) == 0 {
		str := "rpc.Register: type " + sname + " has no exported methods of suitable type"
		log.Print(str)
		return errors.New(str)
	}

	server.serviceMap[s.name] = s
	return nil
}

func (server *Server) CallServiceMethod(codec ServerCodec) {
	// Read the request 
	req, serr := codec.ReadRequest()
	if serr != nil {
		codec.WriteResponse(req, nil, serr)
		return
	}

	// Look up the request.
	server.mu.RLock()
	service := server.serviceMap[req.ServiceName()]
	server.mu.RUnlock()

	if service == nil {
		msg := "rpc: can't find service " + req.ServiceName()
		codec.WriteResponse(req, nil, NewServerError(ERR_INVALID_REQ, msg, nil))
		return
	}

	serviceMethod := service.method[req.MethodName()]
	if serviceMethod == nil {
		msg := "rpc: can't find method " + req.MethodName()
		codec.WriteResponse(req, nil, NewServerError(ERR_NO_METHOD, msg, nil))
		return
	}

	// Decode the args.
	args := reflect.New(serviceMethod.argsType)
	req.DecodeParams(args.Interface())

	// Call the service method.
	reply := reflect.New(serviceMethod.replyType).Elem()

	function := serviceMethod.method.Func

	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{service.rcvr, args, reply,})

	errInter := returnValues[0].Interface()

	// The return value for the method is an error.
	if errInter != nil {
		serr = NewServerError(ERR_USER_SERVICE, errInter.(error).Error(), nil)
	}

	if err := codec.WriteResponse(req, reply, serr); err != nil {
		log.Fatal(err)
	}

	codec.Close()
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

//-----------------------------------------------------------------------------
// ServerHTTP
//-----------------------------------------------------------------------------

// RPC Server with HTTP connections
type ServerHTTP struct {
	*Server

	codecCreator ServerCodecHTTPCreator
}

// Create a new RPC server with a HTTP connection
func NewServerHTTP(codecCreator ServerCodecHTTPCreator) *ServerHTTP {
	return &ServerHTTP{
		Server: NewServer(),
		codecCreator: codecCreator,
	}
}

func (s *ServerHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "rpc: POST method required, received " + r.Method, http.StatusMethodNotAllowed)
		return
	}

	// http.Error(w, http.StatusText(http.StatusMethodNotAllowed))

	log.Println("new connection established")

	//go s.CallServiceMethod(s.codecCreator.New(w, r))
	s.CallServiceMethod(s.codecCreator.New(w, r))
}

// HandleHTTP registers an HTTP handler for RPC messages on rpcPath.
func (s *ServerHTTP) HandleHTTP(rpcPath string) {
	http.Handle(rpcPath, s)
}

// ListenAndServe accepts incoming HTTP connections on the specified
// address.
func (s *ServerHTTP) ListenAndServe(addr string) error {
	log.Println("waiting for connection...")
	return http.ListenAndServe(addr, nil)
}
