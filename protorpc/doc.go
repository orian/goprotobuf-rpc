// Copyright 2013 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package protorpc implements a Protobuf-RPC ClientCodec and ServerCodec
	for the rpc package.

	Here is a simple proto file("arith.pb/arith.proto"):

		package arith;

		// At least one of xx_generic_services is true
		option cc_generic_services = true;
		option java_generic_services = true;
		option py_generic_services = true;

		message ArithRequest {
			optional int32 a = 1;
			optional int32 b = 2;
		}

		message ArithResponse {
			optional int32 val = 1;
			optional int32 quo = 2;
			optional int32 rem = 3;
		}

		service ArithService {
			rpc multiply (ArithRequest) returns (ArithResponse);
			rpc divide (ArithRequest) returns (ArithResponse);
		}

	Then use "protoc-gen-go" to generate "arith.pb.go" file(include rpc stub):

		$ go install encoding/protobuf/protoc-gen-go
		$ cd arith.pb && protoc --go_out=. arith.proto

	The server calls (for TCP service):

		package server

		import (
			"encoding/protobuf/proto"
			"errors"

			"./arith.pb"
		)

		type Arith int

		func (t *Arith) Multiply(args *arith.ArithRequest, reply *arith.ArithResponse) error {
			reply.Val = proto.Int32(args.GetA() * args.GetB())
			return nil
		}

		func (t *Arith) Divide(args *arith.ArithRequest, reply *arith.ArithResponse) error {
			if args.GetB() == 0 {
				return errors.New("divide by zero")
			}
			reply.Quo = proto.Int32(args.GetA() / args.GetB())
			reply.Rem = proto.Int32(args.GetA() % args.GetB())
			return nil
		}

		func main() {
			arith.ListenAndServeArithService("tcp", ":1984", new(Arith))
		}

	At this point, clients can see a service "Arith" with methods "Arith.Multiply" and
	"Arith.Divide". To invoke one, a client first dials the server:

		client, stub, err := arith.DialArithService("tcp", "127.0.0.1:1984")
		if err != nil {
			log.Fatal(`arith.DialArithService("tcp", "127.0.0.1:1984"):`, err)
		}
		defer client.Close()

	Then it can make a remote call with stub:

		var args ArithRequest
		var reply ArithResponse

		args.A = proto.Int32(7)
		args.B = proto.Int32(8)
		if err = stub.Multiply(&args, &reply); err != nil {
			log.Fatal("arith error:", err)
		}
		fmt.Printf("Arith: %d*%d=%d", args.GetA(), args.GetB(), reply.GetVal())

	Is very simple to use "Protobuf-RPC" with "protoc-gen-go" tool. Try it out.
*/
package protorpc
