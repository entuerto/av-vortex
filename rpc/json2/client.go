// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json2

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/entuerto/av-vortex/rpc"
)

var (
	ErrJsonDecoder = errors.New("Could not create JSON decoder")
)

//-----------------------------------------------------------------------------
// srvRequest
//-----------------------------------------------------------------------------

type clientRequest struct {
	Version string       `json:"jsonrpc"`
	Method  string       `json:"method"`
	Params  interface{}  `json:"params"`
	Id      uint64       `json:"id"`
}

//-----------------------------------------------------------------------------
// srvRequest
//-----------------------------------------------------------------------------

type clientResponse struct {
	Version string            `json:"jsonrpc"`
	Id      uint64            `json:"id"`
	Result  *json.RawMessage  `json:"result"`
	Error   *json.RawMessage  `json:"error"`
}

//-----------------------------------------------------------------------------
// Client HTTP 
//-----------------------------------------------------------------------------

type client struct {
	remoteURL *url.URL
	c *http.Client

	queue chan *rpc.CallResult

	mutex sync.Mutex
	seq   uint64
} 

func (c *client) encodeClientRequest(creq *clientRequest) (io.Reader, error) {
	buf, err := json.Marshal(creq)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(buf), nil
}

func (c *client) decodeServerResponse(resp *http.Response, cres *rpc.CallResult) error {
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	if dec == nil {
		return ErrJsonDecoder
	}

	var cresp clientResponse
	if err := dec.Decode(&cresp); err != nil {
		return err
	}

	if cresp.Error != nil {
		return json.Unmarshal(*cresp.Error, cres.Error)
	}
	return json.Unmarshal(*cresp.Result, cres.Reply)	
}
 
func (c *client) sender() {
	for {
		call := <- c.queue

		c.mutex.Lock()
		c.seq++

		creq := &clientRequest{
			Version: "2.0",
			Method:  call.ServiceMethod,
			Params:  call.Args,
			Id:      uint64(c.seq),
		}

		c.mutex.Unlock()

		body, err := c.encodeClientRequest(creq)
		if err != nil {
			call.Error = err
			call.Done <- call
			continue
		}

		req, err := http.NewRequest("POST", c.remoteURL.String(), body) 
		if err != nil {
			call.Error = err
			call.Done <- call
			continue
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		// Callers should close resp.Body when done reading from it.
		if resp, err := c.c.Do(req); err != nil {
			call.Error = err
		} else {
			call.Error = c.decodeServerResponse(resp, call)
		}

		call.Done <- call
	}
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (c *client) Call(serviceMethod string, args, reply interface{})  *rpc.CallResult {

	result := new(rpc.CallResult)
	result.ServiceMethod = serviceMethod
	result.Args = args
	result.Reply = reply
	result.Done = make(chan *rpc.CallResult)

	c.queue <- result

	return result
}
	
// Close the connection
func (c *client) Close() error {
	return nil
}

// DialHTTP connects to an HTTP RPC-JSON2 server
// at the specified network address and path.
func NewClientHTTP(address, path string) rpc.Client {
	u, err := url.Parse(address + path)
	if err != nil {
		log.Fatal(err)
	}
	
	httpClient:= &client{
		remoteURL: u,
		c: &http.Client{},
		queue: make(chan *rpc.CallResult),
	}

	go httpClient.sender()

	return httpClient
}
