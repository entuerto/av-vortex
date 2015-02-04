// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc


type Client struct {}

// Call invokes the named function, waits for it to complete, and returns its error status.
 func (client *Client) Call(serviceMethod string, args interface{}, reply interface{}) error {

 }

 func (client *Client) Close() error {
 	
 }