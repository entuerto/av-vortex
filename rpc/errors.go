// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import "fmt"

//-----------------------------------------------------------------------------
// ServerError
//-----------------------------------------------------------------------------

type ErrorCode int

const (
	ERR_NONE          ErrorCode = iota
	ERR_PARSE        
	ERR_INVALID_REQ 
	ERR_NO_METHOD   
	ERR_BAD_PARAMS  
	ERR_INTERNAL    
	ERR_SERVER   
	ERR_USER_SERVICE  ErrorCode = iota + 1000
)
// ServerError represents an error that has been returned from
// the remote side of the RPC connection.
type ServerError struct {
	Code    ErrorCode
	Message string   
	Data    interface{} 
}

func (e ServerError) Error() string {
	return fmt.Sprintf("Error: (%v), %s", e.Code, e.Message)
}

//-----------------------------------------------------------------------------
// ServerErrorCreator
//-----------------------------------------------------------------------------
//type ServerErrorCreator interface {
func NewServerError(code ErrorCode, msg string, d interface{}) *ServerError {
	return &ServerError{
		Code: code,
		Message: msg,
		Data: d,
	}
}