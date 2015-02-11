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
	"runtime"
	"errors"
	"net/http"

	"github.com/golang/glog"
 	"github.com/entuerto/av-vortex/rpc"
	"github.com/entuerto/av-vortex/rpc/json2"
)

type ServerInfo struct {
	Cpus int
	Mem  runtime.MemStats
}

func (si *ServerInfo) ServerStats(param interface{}, reply *ServerInfo) error {
	glog.Infof("ServerStats...\n")

	if reply == nil {
		return errors.New("reply nil")
	}
	reply.Cpus = runtime.NumCPU()
	runtime.ReadMemStats(&reply.Mem)
	return nil
}

type Args struct {
	A, B int
}

type Calculator int

func (c Calculator) Add(args *Args, reply *int) error {
	glog.Infof("Add ( %d + %d )\n", args.A, args.B)

	*reply = args.A + args.B
	return nil
}

func main() {
	var (
		addr = flag.String("addr", ":5000", "Address/port to listen on")
	)

	// Parse the command-line flags.
	flag.Parse()

	si := new(ServerInfo)
	ca := new(Calculator)

 	server := rpc.NewServer()

	if err := server.Register(si); err != nil {
		glog.Error(err)
	}

	if err := server.Register(ca); err != nil {
		glog.Error(err)
	}

	json2.HandleHTTP("/rpc", server)

	glog.Infoln("Waiting for connection...")

	glog.Fatal(http.ListenAndServe(*addr, nil))
}
