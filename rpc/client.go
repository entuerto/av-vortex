// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

type CallResult struct {
	ServiceMethod string      // The name of the service and method to call.
	Args          interface{}
	Reply         interface{} // The reply from the RPC server
 	Error         error       // After completion, the error status.
	Done          chan *CallResult  
}

type Client interface {
	// Call invokes the named function, waits for it to complete, and returns its error status.
	Call(serviceMethod string, args, reply interface{}) *CallResult
	// Close the connection
	Close() error
}
