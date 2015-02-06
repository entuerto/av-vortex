// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package rpc provides access to the exported methods of an object across a
network or other I/O connection.  A server registers an object, making it visible
as a service with the name of the type of the object.  After registration, exported
methods of the object will be accessible remotely.  A server may register multiple
objects (services) of different types but it is an error to register multiple
objects of the same type.

Only methods that satisfy these criteria will be made available for remote access;
other methods will be ignored:

	- the method is exported.
	- the method has two arguments, both exported (or builtin) types.
	- the method's second argument is a pointer.
	- the method has return type error.

In effect, the method must look schematically like

	func (t *T) MethodName(argType T1, replyType *T2) error

The method's first argument represents the arguments provided by the caller; the
second argument represents the result parameters to be returned to the caller.
The method's return value, if non-nil, is passed back as a string that the client
sees as if created by errors.New.  If an error is returned, the reply parameter
will not be sent back to the client.
*/
package rpc

import (
	"errors"
	"flag"
	"reflect"
	"sync"
)

// Client request. When the client sends a request it is in
// the format "Service.Method"
type Request interface {
	ServiceName() string      
	MethodName()  string    
	DecodeParams(interface{}) error 
	Result() chan *Result 
}

// Result from the specified request
type Result struct {
	Value  interface{}
	Error  error
}

// Returns a new result structure
func NewResult(value interface{}, err error) *Result {
	return &Result{
		Value: value,
		Error: err,
	}
}

//-----------------------------------------------------------------------------
// Server
//-----------------------------------------------------------------------------

var nWorkers = flag.Int("w", 10, "Number of workers")

// Server represents an RPC Server.
type Server struct {
	mu          sync.RWMutex         // protects the serviceMap
	serviceMap  ServiceMap

	RequestQueue chan Request
}

// Return a new RPC server
func NewServer() *Server {
	srv := &Server{
		serviceMap: make(ServiceMap),
	}

	srv.RequestQueue = workerPool(srv, *nWorkers)
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
		return errors.New("RPC: type " + sname + " is not exported")
	}

	if name != "" {
		sname = name
	}

	if _, present := server.serviceMap[sname]; present {
		return errors.New("RPC: service already defined: " + sname)
	}

	s.name = sname
	s.method = installValidMethods(s.typ)

	if len(s.method) == 0 {
		return errors.New("RPC: type " + sname + " has no exported methods of suitable type")
	}

	server.serviceMap[s.name] = s
	return nil
}

// Takes a RPC request and produces result from the specified service
func (server *Server) ServeRequest(req Request) *Result {
	// Look up the request.
	server.mu.RLock()
	service := server.serviceMap[req.ServiceName()]
	server.mu.RUnlock()

	if service == nil {
		ErrMethodNotFound.Data = req.ServiceName() 
		return NewResult(nil, ErrMethodNotFound)
	}

	reply, err := service.Call(req)

	return NewResult(reply, err)
}

//-----------------------------------------------------------------------------
// Workers
//-----------------------------------------------------------------------------

// Initialize a pool of worker goroutines
func workerPool(srv *Server, n int) chan Request {
	requests := make(chan Request)

	for i := 0; i < n; i++ {
		go worker(srv, requests)
	}

	return requests
}

// Worker function serve requests
func worker(srv *Server, requests chan Request) {

	for r := range requests {

		result := srv.ServeRequest(r)

		r.Result() <- result 
	}

}