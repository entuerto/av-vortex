// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json2

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"log"
	_ "sync"
	"strings"

	"github.com/entuerto/av-vortex/rpc"
)

//-----------------------------------------------------------------------------
// Request
//-----------------------------------------------------------------------------
type jsonRequest struct {
	rpc.Request

	serviceName string       `json:"-"`
	methodName  string       `json:"-"`

	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params"`
	Id      *json.RawMessage `json:"id"`
}

func (r jsonRequest) ServiceName() string {
	return r.serviceName
}  

func (r jsonRequest) MethodName() string {
	return r.methodName
}

func (r jsonRequest) DecodeParams(args interface{}) error {
	if args != nil {
		return json.Unmarshal(*r.Params, &args)
	}
	return nil
}

//-----------------------------------------------------------------------------
// Reponse
//-----------------------------------------------------------------------------
type jsonResponse struct {
	Version string           `json:"jsonrpc"`
	Id      *json.RawMessage `json:"id"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *jsonError       `json:"error,omitempty"`
}

//-----------------------------------------------------------------------------
// ServerCodec
//-----------------------------------------------------------------------------
type ServerCodec struct {
	reader io.Reader
	writer io.Writer
}

func (c ServerCodec) ReadRequest() (rpc.Request, *rpc.ServerError) {
	log.Printf("[%p] ReadRequest...\n", c)

	dec := json.NewDecoder(c.reader)
	if dec == nil {
		return nil, rpc.NewServerError(rpc.ERR_INTERNAL, "Could not create JSON decoder", nil)
	}

	jreq := new(jsonRequest)

	if err := dec.Decode(&jreq); err != nil {
		return jreq, rpc.NewServerError(rpc.ERR_INTERNAL, err.Error(), nil)
	}

	if jreq.Version != "2.0" {
		return jreq, rpc.NewServerError(rpc.ERR_INVALID_REQ, "Invalid version", nil)
	}

	// find service
	dot := strings.LastIndex(jreq.Method, ".")
	if dot < 0 {
		return jreq, rpc.NewServerError(rpc.ERR_INVALID_REQ, "rpc: service/method request ill-formed: " + jreq.Method, nil)
	}

	jreq.serviceName = jreq.Method[:dot]
	jreq.methodName  = jreq.Method[dot+1:]
	return jreq, nil  
}

func (c ServerCodec) WriteResponse(request rpc.Request, result interface{}, serr *rpc.ServerError) error {
	log.Printf("[%p]  WriteResponse...\n", c)

	enc := json.NewEncoder(c.writer)
	if enc == nil {
		return errors.New("Could not create JSON encoder")
	} 

	jreq := request.(*jsonRequest)
	if jreq == nil {
		return errors.New("Could not cast to JSON request")
	} 

	jresp := jsonResponse{
		Version: jreq.Version,
		Id: jreq.Id,
	}

	if serr != nil {
		jresp.Error = newJsonErrorFromServerError(serr) 
	} else {
		jresp.Result = result
	}

	return enc.Encode(jresp)
}

func (c ServerCodec) Close() error {
	log.Printf("[%p] Close...\n", c)

	//w := c.writer.(io.WriteCloser)
	//return w.Close()
	return nil
}

func NewServerCodec(w http.ResponseWriter, r *http.Request) *ServerCodec {
	// Prevents Internet Explorer from MIME-sniffing a response away
	// from the declared content-type
	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	return &ServerCodec{
		reader: r.Body,
		writer: w,
	}
}

//-----------------------------------------------------------------------------
// ServerCodecCreator
//-----------------------------------------------------------------------------

// 
type ServerCodecCreator struct {}

// NewServerCodec returns a new rpc.ServerCodec using JSON-RPC
func (c ServerCodecCreator) New(w http.ResponseWriter, r *http.Request) rpc.ServerCodec  {
	return NewServerCodec(w, r)
}
