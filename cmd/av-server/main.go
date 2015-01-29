// Copyright 2015 The av-vortex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

/* test with curl
  curl -X POST -H "Content-Type: application/json" \
       -d '{"method":"HelloService.Say","params":[{"Who":"Test"}], "id":"1"}' \
       http://localhost:10000/rpc
*/

import (
   "flag"
   "log"
   "runtime"
   "errors"
   _ "expvar"

   "github.com/entuerto/av-vortex/rpc"
   "github.com/entuerto/av-vortex/rpc/json2"
)

var addr = flag.String("http", ":5000", "http service address")

type ServerInfo struct {
   Cpus int
   Mem  runtime.MemStats
}

func (si *ServerInfo) ServerStats(param interface{}, reply *ServerInfo) error {
   log.Printf("ServerStats...\n")

   if reply == nil {
      return errors.New("reply nil")
   }
   reply.Cpus = runtime.NumCPU()
   runtime.ReadMemStats(&reply.Mem)
   return nil
}

func main() {
   si := new(ServerInfo)

   server := rpc.NewServerHTTP(new(json2.ServerCodecCreator))
   server.Register(si)
   server.HandleHTTP("/rpc")
   server.ListenAndServe(*addr)
}
