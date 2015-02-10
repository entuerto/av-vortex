// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json2

import (
	"errors"
	"net/http/httptest"
	_ "runtime"
	"sync"
	_ "sync/atomic"
	"testing"

	"github.com/entuerto/av-vortex/rpc"
)

type Args struct {
	A, B int
}

type Reply struct {
	C int
}
//-----------------------------------------------------------------------------

var (
	srv *rpc.Server
	testHttpSrv *httptest.Server
	once sync.Once
)

func startServer() {
	srv = rpc.NewServer()
	srv.Register(new(Arith))

	testHttpSrv = httptest.NewServer(&handler{srv})
}

//-----------------------------------------------------------------------------

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

func TestJson2RPC_GoodCalls(t *testing.T) {
	var args *Args
	var result *rpc.CallResult
	var reply Reply

	once.Do(startServer)

	c := NewClientHTTP(testHttpSrv.URL, "/")

	// Good call
	args = &Args{7, 0}
	result = c.Call("Arith.Add", args, &reply)
	<- result.Done

	// expect an error
	if result.Error != nil {
		t.Errorf("Add: expected no error but got string %q", result.Error.Error())
	}
	if reply, ok := result.Reply.(*Reply); !ok || reply.C != args.A + args.B {
		t.Errorf("Add: expected %d got %d", args.A + args.B, reply.C)
	}


	args = &Args{7, 8}
	result = c.Call("Arith.Mul", args, &reply)
	<- result.Done

	if result.Error != nil {
		t.Errorf("Mul: expected no error but got string %q", result.Error.Error())
	}
	if reply, ok := result.Reply.(*Reply); !ok || reply.C != args.A * args.B {
		t.Errorf("Mul: expected %d got %d", args.A * args.B, reply.C)
	}
}

func TestJson2RPC_MethodNotFound(t *testing.T) {
	var args *Args
	var result *rpc.CallResult
	var reply Reply

	once.Do(startServer)

	c := NewClientHTTP(testHttpSrv.URL, "/")

	// Nonexistent method
	args = &Args{7, 0}
	result = c.Call("Arith.BadOperation", args, &reply)
	<- result.Done

	// expect an error
	if result.Error == nil {
		t.Error("BadOperation: expected error")
	} else if result.Error.Code != rpc.ERR_NO_METHOD {
		t.Errorf("BadOperation: expected can't find method error; got %q", result.Error)
	}
}

func TestJson2RPC_UnknownService(t *testing.T) {
	var args *Args
	var result *rpc.CallResult
	var reply Reply

	once.Do(startServer)

	c := NewClientHTTP(testHttpSrv.URL, "/")

	// Unknown service
	args = &Args{7, 8}
	result = c.Call("Arith.Unknown", args, &reply)
	<- result.Done

	if result.Error == nil {
		t.Error("expected error calling unknown service")
	} else if result.Error.Code != rpc.ERR_NO_METHOD {
		t.Error("expected error about method; got", result.Error)
	}
}

func TestJson2RPC_MethodErrorMessage(t *testing.T) {
	var args *Args
	var result *rpc.CallResult
	var reply Reply

	once.Do(startServer)

	c := NewClientHTTP(testHttpSrv.URL, "/")

	// Error test
	args = &Args{7, 0}
	result = c.Call("Arith.Div", args, &reply)
	<- result.Done

	// expect an error: zero divide
	if result.Error == nil {
		t.Error("Div: expected error")
	} else if result.Error.Message != "divide by zero" {
		t.Error("Div: expected divide by zero error; got", result.Error)
	}
}

func BenchmarkServeRequest(b *testing.B) {
	once.Do(startServer)

	// Good call
	args := &Args{7, 0}
	b.ResetTimer()

	var reply Reply

	for i := 0; i < b.N; i++ {
		c := NewClientHTTP(testHttpSrv.URL, "/")
 	
		result := c.Call("Arith.Add", args, &reply)
		<- result.Done
	
		// expect an error
		if result.Error != nil {
			b.Fatalf("Add: expected no error but got string %q", result.Error.Error())
		}
		if reply, ok := result.Reply.(*Reply); !ok || reply.C != args.A + args.B {
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
		var reply Reply
		for pb.Next() {
			c := NewClientHTTP(testHttpSrv.URL, "/")
 		
			result := c.Call("Arith.Add", args, &reply)
			<- result.Done
		
			// expect an error
			if result.Error != nil {
				b.Fatalf("Add: expected no error but got string %q", result.Error.Error())
			}
			if reply, ok := result.Reply.(*Reply); !ok || reply.C != args.A + args.B {
				b.Fatalf("Add: expected %d got %d", args.A + args.B, reply.C)
			}
		}	
	})

}
/*
func BenchmarkServeRequestAsync(b *testing.B) {
	const MaxConcurrentCalls = 100
	once.Do(startServer)

	args  := &Args{7, 0}
	procs := 4 * runtime.GOMAXPROCS(-1)
	send  := int32(b.N)
	recv  := int32(b.N)

	gate := make(chan bool, MaxConcurrentCalls)
	//res := make(chan *rpc.CallResult, MaxConcurrentCalls)

	c := NewClientHTTP(testHttpSrv.URL, "/")

	var (
		result *rpc.CallResult
		wg sync.WaitGroup
	)
	wg.Add(procs)

	b.ResetTimer()

	for p := 0; p < procs; p++ {
		go func() {
			for atomic.AddInt32(&send, -1) >= 0 {
				gate <- true

				reply := new(Reply)
				result = c.Call("Arith.Add", args, reply)
			}
		}()
		go func() {
			for call := range result.Done {
				A := call.Args.(*Args).A
				B := call.Args.(*Args).B
				C := call.Reply.(*Reply).C
				if A+B != C {
					b.Fatalf("incorrect reply: Add: expected %d got %d", A+B, C)
				}
				<-gate
				if atomic.AddInt32(&recv, -1) == 0 {
					close(result.Done)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
*/