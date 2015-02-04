// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json2

import (
	"fmt"

	"github.com/entuerto/av-vortex/rpc"
)

type jsonErrorCode int

// The error codes from and including -32768 to -32000 are reserved for pre-defined errors.
const (
	JSON_ERR_PARSE       jsonErrorCode = -32700
	JSON_ERR_INVALID_REQ jsonErrorCode = -32600
	JSON_ERR_NO_METHOD   jsonErrorCode = -32601
	JSON_ERR_BAD_PARAMS  jsonErrorCode = -32602
	JSON_ERR_INTERNAL    jsonErrorCode = -32603
	JSON_ERR_SERVER      jsonErrorCode = -32000
)

type jsonError struct {
	// A Number that indicates the error type that occurred.
	Code jsonErrorCode `json:"code"`    /* required */

	// A String providing a short description of the error.
	// The message SHOULD be limited to a concise single sentence.
	Message string     `json:"message"` /* required */

	// A Primitive or Structured value that contains additional information about the error.
	Data interface{}   `json:"data"`    /* optional */
}

func (e jsonError) Error() string {
	return fmt.Sprintf("Error: (%v), %s", e.Code, e.Message)
}

// Returns a new jsonError for JSON-RPC
func newJsonError(code jsonErrorCode, message string, data interface{}) *jsonError {
	return &jsonError{
		Code: code,
		Message: message,
		Data: data,
	}
}

// Creates a Json Error from a RPC server error
func newJsonErrorFromError(err error) *jsonError {
	var jcode jsonErrorCode

	serr, ok := err.(*rpc.ServerError)
	if !ok {
		return nil
	}

	switch serr.Code {
		case rpc.ERR_PARSE:
			jcode = JSON_ERR_PARSE        
		case rpc.ERR_INVALID_REQ:
			jcode = JSON_ERR_INVALID_REQ
		case rpc.ERR_NO_METHOD:
			jcode = JSON_ERR_NO_METHOD   
		case rpc.ERR_BAD_PARAMS:
			jcode = JSON_ERR_BAD_PARAMS  
		case rpc.ERR_INTERNAL:
			jcode = JSON_ERR_INTERNAL    
		case rpc.ERR_SERVER:
			jcode = JSON_ERR_SERVER 
		default:
			jcode = jsonErrorCode(serr.Code)
	}

	return newJsonError(jcode, serr.Message, serr.Data)
}

// Creates a Json Error from a RPC server error
func newJsonErrorFromServerError(serr *rpc.ServerError) *jsonError {
	var jcode jsonErrorCode

	switch serr.Code {
		case rpc.ERR_PARSE:
			jcode = JSON_ERR_PARSE        
		case rpc.ERR_INVALID_REQ:
			jcode = JSON_ERR_INVALID_REQ
		case rpc.ERR_NO_METHOD:
			jcode = JSON_ERR_NO_METHOD   
		case rpc.ERR_BAD_PARAMS:
			jcode = JSON_ERR_BAD_PARAMS  
		case rpc.ERR_INTERNAL:
			jcode = JSON_ERR_INTERNAL    
		case rpc.ERR_SERVER:
			jcode = JSON_ERR_SERVER 
		default:
			jcode = jsonErrorCode(serr.Code)
	}

	return newJsonError(jcode, serr.Message, serr.Data)
}
