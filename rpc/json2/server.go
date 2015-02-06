// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json2

import (
	"encoding/json"
	_ "errors"
	"io"
	"log"	
	"net/http"
	_ "sync"
	"strings"

	"github.com/entuerto/av-vortex/rpc"
)

//-----------------------------------------------------------------------------
// srvRequest
//-----------------------------------------------------------------------------

type srvRequest struct {
	rpc.Request

	result chan *rpc.Result

	serviceName string       `json:"-"`
	methodName  string       `json:"-"`

	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params"`
	Id      *json.RawMessage `json:"id"`
}

func (r srvRequest) ServiceName() string {
	return r.serviceName
}  

func (r srvRequest) MethodName() string {
	return r.methodName
}

func (r srvRequest) DecodeParams(args interface{}) error {
	if args != nil {
		return json.Unmarshal(*r.Params, &args)
	}
	return nil
}

func (r srvRequest) Result() chan *rpc.Result {
	return r.result
}

func newRequest() *srvRequest {
	return &srvRequest{
		result: make(chan *rpc.Result),
	}
}

//-----------------------------------------------------------------------------
// Reponse
//-----------------------------------------------------------------------------

type srvResponse struct {
	Version string            `json:"jsonrpc"`
	Id      *json.RawMessage  `json:"id"`
	Result  interface{}       `json:"result,omitempty"`
	Error   *jsonError        `json:"error,omitempty"`
}


func readRequest(reader io.ReadCloser) (rpc.Request, error) {
	log.Printf("[%p] ReadRequest...\n", reader)
	defer reader.Close()

	dec := json.NewDecoder(reader)
	if dec == nil {
		rpc.ErrInternal.Message = "Could not create JSON decoder"
		return nil, rpc.ErrInternal
	}

	jreq := newRequest()

	if err := dec.Decode(&jreq); err != nil {
		rpc.ErrInternal.Message = err.Error()
		return jreq, rpc.ErrInternal 
	}

	if jreq.Version != "2.0" {
		rpc.ErrInvalidRequest.Message = "RPC-JSON2: Invalid version"
		return jreq, rpc.ErrInvalidRequest 
	}

	// find service
	dot := strings.LastIndex(jreq.Method, ".")
	if dot < 0 {
		rpc.ErrInvalidRequest.Message = "RPC-JSON2: service/method request ill-formed" 
		rpc.ErrInvalidRequest.Data = jreq.Method
		return jreq, rpc.ErrInvalidRequest 
	}

	jreq.serviceName = jreq.Method[:dot]
	jreq.methodName  = jreq.Method[dot+1:]
	return jreq, nil  
}

func writeResponse(writer io.Writer, request rpc.Request, result *rpc.Result) error {
	log.Printf("[%p]  WriteResponse...\n", writer)

	enc := json.NewEncoder(writer)
	if enc == nil {
		rpc.ErrInternal.Message = "Could not create JSON encoder"
		return rpc.ErrInternal 
	} 

	jreq, ok := request.(*srvRequest)
	if !ok {
		rpc.ErrInternal.Message = "Could not cast to JSON request"
		return rpc.ErrInternal 
	} 

	jresp := srvResponse{
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
		http.Error(w, "RPC-JSON2: POST method required, received " + r.Method, http.StatusMethodNotAllowed)
		return
	}

	log.Println("New connection established")
	
	var result *rpc.Result

	request, err := readRequest(r.Body)

	if err == nil {
	    h.RequestQueue <- request
    	result = <-request.Result() // this blocks
	} else {
		result = rpc.NewResult(nil, err)
	}

	// Prevents Internet Explorer from MIME-sniffing a response away
	// from the declared content-type
	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := writeResponse(w, request, result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
