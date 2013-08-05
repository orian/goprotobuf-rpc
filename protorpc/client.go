// Copyright 2013 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protorpc

import (
	"bufio"
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	"github.com/orian/goprotobuf-rpc/protorpc/wire.pb"
	"io"
	"net"
	"net/rpc"
	"sync"
)

type clientCodec struct {
	r *bufio.Reader
	w io.Writer
	c io.Closer

	// Protobuf-RPC responses include the request id but not the request method.
	// Package rpc expects both.
	// We save the request method in pending when sending a request
	// and then look it up by request ID when filling out the rpc Response.
	mutex   sync.Mutex        // protects pending
	pending map[uint64]string // map request id to method name
}

// NewClientCodec returns a new rpc.ClientCodec using Protobuf-RPC on conn.
func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &clientCodec{
		r:       bufio.NewReader(conn),
		w:       conn,
		c:       conn,
		pending: make(map[uint64]string),
	}
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	var pb proto.Message
	if param != nil {
		var ok bool
		if pb, ok = param.(proto.Message); !ok {
			return fmt.Errorf("ClientCodec.WriteRequest: %T does not implement proto.Message", param)
		}
	}

	c.mutex.Lock()
	c.pending[r.Seq] = r.ServiceMethod
	c.mutex.Unlock()

	var header wire.Header
	header.Method = proto.String(r.ServiceMethod)
	header.Id = proto.Uint64(r.Seq)

	// Write the Header and Param
	if err := WriteProto(c.w, &header); err != nil {
		return err
	}
	return WriteProto(c.w, pb)
}

func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	var header wire.Header
	if err := ReadProto(c.r, &header); err != nil {
		return err
	}

	c.mutex.Lock()
	r.ServiceMethod = c.pending[header.GetId()]
	delete(c.pending, header.GetId())
	c.mutex.Unlock()

	r.Error = ""
	r.Seq = header.GetId()
	if header.Error != nil {
		r.Error = header.GetError()
	}
	return nil
}

func (c *clientCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	pb, ok := x.(proto.Message)
	if !ok {
		return fmt.Errorf("ClientCodec.ReadResponseBody: %T does not implement proto.Message", x)
	}
	return ReadProto(c.r, pb)
}

// Close closes the underlying connection.
func (c *clientCodec) Close() error {
	return c.c.Close()
}

// NewClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(conn))
}

// Dial connects to a JSON-RPC server at the specified network address.
func Dial(network, address string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn), err
}
