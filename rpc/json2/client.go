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

// Call invokes the named function, waits for it to complete, and returns its error status.
func (c *client) Call(serviceMethod string, args, reply interface{})  *rpc.CallResult {

	c.mutex.Lock()
	c.seq++

	creq := &clientRequest{
		Version: "2.0",
		Method:  serviceMethod,
		Params:  args,
		Id:      uint64(c.seq),
	}

	c.mutex.Unlock()

	result := new(rpc.CallResult)
	result.Reply = reply
	
	body, err := c.encodeClientRequest(creq)
	if err != nil {
		result.Error = err
		return result
	}

	req, err := http.NewRequest("POST", c.remoteURL.String(), body) 
	if err != nil {
		result.Error = err
		return result
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// Callers should close resp.Body when done reading from it.
	if resp, err := c.c.Do(req); err != nil {
		result.Error = err
	} else {
		result.Error = c.decodeServerResponse(resp, result)
	}

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
	
	return &client{
		remoteURL: u,
		c: &http.Client{},
	}
}
