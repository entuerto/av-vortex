// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"sync"
	"testing"
)

type Args struct {
	A, B int
}

type Reply struct {
	C int
}
//-----------------------------------------------------------------------------

var (
	srv *Server
	once sync.Once
)

func startServer() {
	srv = NewServer()
	srv.Register(new(Arith))
}

//-----------------------------------------------------------------------------

type ReplyNotPointer int
type ArgNotPublic int
type ReplyNotPublic int
type NeedsPtrType int
type local struct{}

func (t *ReplyNotPointer) ReplyNotPointer(args *Args, reply Reply) error {
	return nil
}

func (t *ArgNotPublic) ArgNotPublic(args *local, reply *Reply) error {
	return nil
}

func (t *ReplyNotPublic) ReplyNotPublic(args *Args, reply *local) error {
	return nil
}

func (t *NeedsPtrType) NeedsPtrType(args *Args, reply *Reply) error {
	return nil
}

// Check that registration handles lots of bad methods and a type with no suitable methods.
func TestRegistrationError(t *testing.T) {
	once.Do(startServer) 

	err := srv.Register(new(ReplyNotPointer))
	if err == nil {
		t.Error("expected error registering ReplyNotPointer")
	}
	err = srv.Register(new(ArgNotPublic))
	if err == nil {
		t.Error("expected error registering ArgNotPublic")
	}
	err = srv.Register(new(ReplyNotPublic))
	if err == nil {
		t.Error("expected error registering ReplyNotPublic")
	}
	err = srv.Register(NeedsPtrType(0))
	if err == nil {
		t.Error("expected error registering NeedsPtrType")
	} 
}

//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// testRequest
//-----------------------------------------------------------------------------

type testRequest struct {
	serviceName string      
	methodName  string  
	args Args    

	result chan *Result
}

func (r testRequest) ServiceName() string {
	return r.serviceName
}  

func (r testRequest) MethodName() string {
	return r.methodName
}

func (r testRequest) DecodeParams(args interface{}) error {
	if a, ok := args.(*Args); ok {
		*a = r.args
	}
	return nil
}

func (r testRequest) Result() chan *Result {
	return r.result
}

func newTestRequest(s, m string, args *Args) *testRequest {
	return &testRequest{
		serviceName: s,
		methodName: m, 
		args: *args,
		result: make(chan *Result),
	}
}

//-----------------------------------------------------------------------------

type Arith int

// Some of Arith's methods have value args, some have pointer args. That's deliberate.

func (t *Arith) Add(args Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

func (t *Arith) Mul(args *Args, reply *Reply) error {
	reply.C = args.A * args.B
	return nil
}

func (t *Arith) Div(args Args, reply *Reply) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	reply.C = args.A / args.B
	return nil
}

func TestRPC_GoodCalls(t *testing.T) {
	var args *Args
	var req *testRequest
	var result *Result

	once.Do(startServer)

	// Good call
	args = &Args{7, 0}
	req = newTestRequest("Arith", "Add", args)
	result = srv.ServeRequest(req) 
	// expect an error
	if result.Error != nil {
		t.Errorf("Add: expected no error but got string %q", result.Error.Error())
	}
	if reply, ok := result.Value.(*Reply); !ok || reply.C != args.A + args.B {
		t.Errorf("Add: expected %d got %d", args.A + args.B, reply.C)
	}

	args = &Args{7, 8}
	req = newTestRequest("Arith", "Mul", args)
	result = srv.ServeRequest(req)
	if result.Error != nil {
		t.Errorf("Mul: expected no error but got string %q", result.Error.Error())
	}
	if reply, ok := result.Value.(*Reply); !ok || reply.C != args.A * args.B {
		t.Errorf("Mul: expected %d got %d", args.A * args.B, reply.C)
	}
}

func TestRPC_MethodNotFound(t *testing.T) {
	var args *Args
	var req *testRequest
	var result *Result

	once.Do(startServer)

	// Nonexistent method
	args = &Args{7, 0}
	req = newTestRequest("Arith", "BadOperation", args)
	result = srv.ServeRequest(req) 
	// expect an error
	if result.Error == nil {
		t.Error("BadOperation: expected error")
	} else if result.Error != ErrMethodNotFound {
		t.Errorf("BadOperation: expected can't find method error; got %q", result.Error)
	}
}

func TestRPC_UnknownService(t *testing.T) {
	var args *Args
	var req *testRequest
	var result *Result

	once.Do(startServer)

	// Unknown service
	args = &Args{7, 8}
	req = newTestRequest("Arith", "Unknown", args)
	result = srv.ServeRequest(req)
	if result.Error == nil {
		t.Error("expected error calling unknown service")
	} else if result.Error != ErrMethodNotFound {
		t.Error("expected error about method; got", result.Error)
	}
}

func TestRPC_MethodErrorMessage(t *testing.T) {
	var args *Args
	var req *testRequest
	var result *Result

	once.Do(startServer)

	// Error test
	args = &Args{7, 0}
	req = newTestRequest("Arith", "Div", args)
	result = srv.ServeRequest(req)
	// expect an error: zero divide
	if result.Error == nil {
		t.Error("Div: expected error")
	} else if result.Error.Error() != "divide by zero" {
		t.Error("Div: expected divide by zero error; got", result.Error)
	}

}

func BenchmarkServeRequest(b *testing.B) {
	once.Do(startServer)

	// Good call
	args := &Args{7, 0}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := newTestRequest("Arith", "Add", args)
		result := srv.ServeRequest(req) 
		// expect an error
		if result.Error != nil {
			b.Fatalf("Add: expected no error but got string %q", result.Error.Error())
		}
		if reply, ok := result.Value.(*Reply); !ok || reply.C != args.A + args.B {
			b.Fatalf("Add: expected %d got %d", args.A + args.B, reply.C)
		}	
	}
}

func BenchmarkServeRequestParallel(b *testing.B) {
	once.Do(startServer)

	// Good call
	args := &Args{7, 0}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := newTestRequest("Arith", "Add", args)
			result := srv.ServeRequest(req) 
			// expect an error
			if result.Error != nil {
				b.Fatalf("Add: expected no error but got string %q", result.Error.Error())
			}
			if reply, ok := result.Value.(*Reply); !ok || reply.C != args.A + args.B {
				b.Fatalf("Add: expected %d got %d", args.A + args.B, reply.C)
			}
		}	
	})
}
