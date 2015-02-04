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

	result chan *rpc.Result

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

func (r jsonRequest) Result() chan *rpc.Result {
	return r.result
}

func newJsonRequest() *jsonRequest {
	return &jsonRequest{
		result: make(chan *rpc.Result),
	}
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

func ReadRequest(reader io.Reader) (rpc.Request, error) {
	log.Printf("[%p] ReadRequest...\n", reader)

	dec := json.NewDecoder(reader)
	if dec == nil {
		return nil, rpc.NewServerError(rpc.ERR_INTERNAL, "Could not create JSON decoder", nil)
	}

	jreq := newJsonRequest()

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

func WriteResponse(writer io.Writer, request rpc.Request, result *rpc.Result) error {
	log.Printf("[%p]  WriteResponse...\n", writer)

	enc := json.NewEncoder(writer)
	if enc == nil {
		return errors.New("Could not create JSON encoder")
	} 

	jreq, ok := request.(*jsonRequest)
	if !ok {
		return errors.New("Could not cast to JSON request")
	} 

	jresp := jsonResponse{
		Version: jreq.Version,
		Id: jreq.Id,
	}

	if result.Error != nil {
		jresp.Error = newJsonErrorFromError(result.Error) 
	} else {
		jresp.Result = result.Value
	}

	return enc.Encode(jresp)
}

func (c ServerCodec) Close() error {
	log.Printf("[%p] Close...\n", c)

	//w := c.writer.(io.WriteCloser)
	//return w.Close()
	return nil
}

//-----------------------------------------------------------------------------
// Handle HTTP requests
//-----------------------------------------------------------------------------

func HandleHTTP(path string, srv *rpc.Server) {
	http.Handle(path, &handler{srv})
}

type handler struct {
	*rpc.Server
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "rpc: POST method required, received " + r.Method, http.StatusMethodNotAllowed)
		return
	}

	log.Println("New connection established")
/*
	codec := &ServerCodec{
		reader: r.Body,
		writer: w,
	}
*/
	
	request, _ := ReadRequest(r.Body)

    h.RequestQueue <- request
    result := <-request.Result() // this blocks
	
	// Prevents Internet Explorer from MIME-sniffing a response away
	// from the declared content-type
	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	//go h.CallServiceMethod(codec)

	WriteResponse(w, request, result)
}
