// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import "fmt"

// The error codes from and including -32768 to -32000 are reserved for pre-defined errors.
const (
	ERR_PARSE       = -32700
	ERR_INVALID_REQ = -32600
	ERR_NO_METHOD   = -32601
	ERR_BAD_PARAMS  = -32602
	ERR_INTERNAL    = -32603
	ERR_SERVER      = -32000
)

var (
	ErrParser          = NewServerError(ERR_PARSE, "An error occurred on the server while parsing the RPC request.", nil)
	ErrInvalidRequest  = NewServerError(ERR_INVALID_REQ, "Not a valid Request object.", nil)
	ErrMethodNotFound  = NewServerError(ERR_NO_METHOD, "The method does not exist / is not available.", nil)
	ErrInvalidParams   = NewServerError(ERR_BAD_PARAMS, "Invalid method parameter(s).", nil)
	ErrInternal        = NewServerError(ERR_INTERNAL, "Internal RPC error.", nil)
	ErrServer          = NewServerError(ERR_SERVER, "", nil)
)
//-----------------------------------------------------------------------------
// ServerError
//-----------------------------------------------------------------------------

// ServerError represents an error that has been returned from
// the remote side of the RPC connection.
type ServerError struct {
	Code    int
	Message string   
	Data    interface{} 
}

func (e ServerError) Error() string {
	return fmt.Sprintf("Error: (%v), %s", e.Code, e.Message)
}

//-----------------------------------------------------------------------------
// ServerErrorCreator
//-----------------------------------------------------------------------------

// Create ServerError objects
func NewServerError(code int, msg string, d interface{}) *ServerError {
	return &ServerError{
		Code: code,
		Message: msg,
		Data: d,
	}
}

// Format message for ServerError
func FmtServerErrorMessage(svrError *ServerError, value interface{}) *ServerError {
	svrError.Message = fmt.Sprintf(svrError.Message, value) 
	return svrError
}