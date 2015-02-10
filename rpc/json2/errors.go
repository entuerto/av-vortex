// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json2

import (
	"encoding/json"
	"fmt"

	"github.com/entuerto/av-vortex/rpc"
)

var null = json.RawMessage([]byte("null"))

type jsonError struct {
	// A Number that indicates the error type that occurred.
	Code int           `json:"code"`    /* required */

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
func newJsonError(code int, message string, data interface{}) *jsonError {
	return &jsonError{
		Code: code,
		Message: message,
		Data: data,
	}
}

// Creates a Json Error from a RPC server error
func newJsonErrorFromError(err error) *jsonError {
	serr, ok := err.(*rpc.ServerError)
	if !ok {
		return newJsonError(rpc.ERR_INTERNAL, err.Error(), null)
	}

	data := serr.Data
	if data == nil {
		data = null
	}

	return newJsonError(serr.Code, serr.Message, data)
}
