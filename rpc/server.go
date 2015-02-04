// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"flag"
	"log"
	_ "net/http"
	"reflect"
	"sync"
	"errors"
)

// Precompute the reflect type for error.  Can't use error directly
// because Typeof takes an empty interface value.  This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// Client request. When the client sends a request it is in
// the format "Service.Method"
type Request interface {
	ServiceName() string      
	MethodName()  string    
	DecodeParams(interface{}) error 
	Result() chan *Result 
}

type Result struct {
	Value  interface{}
	Error  error
}

//-----------------------------------------------------------------------------
// Server
//-----------------------------------------------------------------------------

// Server represents an RPC Server.
type Server struct {
	mu          sync.RWMutex         // protects the serviceMap
	serviceMap  ServiceMap

	RequestQueue chan Request
}

func NewServer() *Server {
	n := flag.Int("w", 10, "Number of workers")

	srv := &Server{
		serviceMap: make(ServiceMap),
	}

	srv.RequestQueue = WorkerPool(srv, *n)
	return srv
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
		server.serviceMap = make(ServiceMap)
	}

	s := new(Service)
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

func (server *Server) ServeRequest(req Request) *Result {
	// Look up the request.
	server.mu.RLock()
	service := server.serviceMap[req.ServiceName()]
	server.mu.RUnlock()

	if service == nil {
		msg := "rpc: can't find service " + req.ServiceName()
		return &Result{
			Value: nil,
			Error: NewServerError(ERR_INVALID_REQ, msg, nil),
		} 
	}

	reply, err := service.Call(req)

	return &Result{
		Value: reply,
		Error: err,
	}
}

func (server *Server) callServiceMethod(codec ServerCodec) {
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

	reply, err := service.Call(req)

	// The return value for the method is an error.
	if err != nil {
		if e, ok := err.(*ServerError); !ok {
			serr = NewServerError(ERR_USER_SERVICE, err.Error(), nil)
		} else {
			serr = e
		}
	}

	if err := codec.WriteResponse(req, reply, serr); err != nil {
		log.Fatal(err)
	}

	codec.Close()
}

//-----------------------------------------------------------------------------
// 
//-----------------------------------------------------------------------------

//
func WorkerPool(srv *Server, n int) chan Request {
	requests := make(chan Request)

	for i := 0; i < n; i++ {
		go Worker(srv, requests)
	}

	return requests
}

//
func Worker(srv *Server, requests chan Request) {

	for r := range requests {

		result := srv.ServeRequest(r)

		r.Result() <- result 
	}

}