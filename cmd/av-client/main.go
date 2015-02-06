// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
   "log"
   "time"

   "github.com/entuerto/av-vortex/rpc"
   "github.com/entuerto/av-vortex/rpc/json2"
)

type Args struct {
	A, B int
}

func main() {
	c := json2.NewClientHTTP("http://localhost:5000", "/rpc")

	time.Sleep(3 * time.Second)

	log.Println("Connecting...")

	var reply int
	args := &Args{2, 3}

	for i := 0; i < 3; i++ {
		res := c.Call("Calculator.Add", args, &reply)

		go func(res *rpc.CallResult) {
			for {
				select {
				case <- res.Done:
					if (res.Error != nil) {
						log.Fatal(res.Error)
					}
					log.Printf("Reply: %d", reply)
					return
				}
			}
		}(res)
	}

	time.Sleep(3 * time.Second)
}