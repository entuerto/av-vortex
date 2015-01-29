// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import "net/http"

// A ServerCodec implements reading of RPC requests and writing of
// RPC responses for the server side of an RPC session.
//
// The server calls ReadRequest to read requests from the connection,
// and it calls WriteResponse to write a response back.
//
// The server calls Close when finished with the connection.
type ServerCodec interface {
//	ServerErrorCreator
	
	// Reads the RPC request.
	ReadRequest() (Request, *ServerError)

	// WriteResponse must be safe for concurrent use by multiple goroutines.
	WriteResponse(Request, interface{}, *ServerError) error

	Close() error
}

// Creator interface to anable a JSON server codec for HTTP streams
type ServerCodecHTTPCreator interface {
	// Creates a server codec
	New(http.ResponseWriter, *http.Request) ServerCodec
}

// A ClientCodec implements writing of RPC requests and
// reading of RPC responses for the client side of an RPC session.
//
// The client calls WriteRequest to write a request to the connection
// and calls ReadResponse to read responses.
//
// The client calls Close when finished with the connection.
type ClientCodec interface {
	// WriteRequest must be safe for concurrent use by multiple goroutines.
	WriteRequest(*Request, interface{}) error

	ReadResponse(*Response, *interface{}) error

	Close() error
}
