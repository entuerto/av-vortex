// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc


type Response struct {
	ServiceMethod string       // echoes that of the Request
	Id            interface{}  // echoes that of the request
	Error         ServerError  // error, if any.
}