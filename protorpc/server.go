// Copyright 2013 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protorpc

import (
	"bufio"
	"encoding/protobuf/proto"
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"net/rpc/protorpc/wire.pb"
	"sync"
)

type serverCodec struct {
	r *bufio.Reader
	w io.Writer
	c io.Closer

	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	mutex   sync.Mutex // protects seq, pending
	seq     uint64
	pending map[uint64]uint64
}

// NewServerCodec returns a serverCodec that communicates with the ClientCodec
// on the other end of the given conn.
func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return &serverCodec{
		r:       bufio.NewReader(conn),
		w:       conn,
		c:       conn,
		pending: make(map[uint64]uint64),
	}
}

func (c *serverCodec) ReadRequestHeader(r *rpc.Request) error {
	var header wire.Header

	if err := readProto(c.r, &header); err != nil {
		return err
	}
	if header.Method == nil {
		return fmt.Errorf("ServerCodec.ReadRequestHeader: header missing method: %s", header)
	}
	if header.Id == nil {
		return fmt.Errorf("ServerCodec.ReadRequestHeader: header missing seq: %s", header)
	}

	r.ServiceMethod = header.GetMethod()

	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = header.GetId()
	r.Seq = c.seq
	c.mutex.Unlock()

	return nil
}

func (c *serverCodec) ReadRequestBody(x interface{}) error {
	pb, ok := x.(proto.Message)
	if !ok {
		return fmt.Errorf("ServerCodec.ReadRequestBody: %T does not implement proto.Message", x)
	}
	return readProto(c.r, pb)
}

// A value sent as a placeholder for the server's response value when the server
// receives an invalid request. It is never decoded by the client since the Response
// contains an error when it is used.
var invalidRequest = struct{}{}

func (c *serverCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	var pb proto.Message
	if x != nil {
		var ok bool
		if pb, ok = x.(proto.Message); !ok {
			if _, ok = x.(struct{}); !ok {
				c.mutex.Lock()
				delete(c.pending, r.Seq)
				c.mutex.Unlock()
				return fmt.Errorf("ServerCodec.WriteResponse: %T does not implement proto.Message", x)
			}
		}
	}

	c.mutex.Lock()
	id, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("ServerCodec.WriteResponse: invalid sequence number in response")
	}
	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	var header wire.Header
	header.Id = proto.Uint64(id)
	if r.Error != "" {
		header.Error = proto.String(r.Error)
	}

	// Write the Header and Result
	if err := writeProto(c.w, &header); err != nil {
		return err
	}
	return writeProto(c.w, pb)
}

func (s *serverCodec) Close() error {
	return s.c.Close()
}

// ServeConn runs the JSON-RPC server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn in a go statement.
func ServeConn(conn io.ReadWriteCloser) {
	rpc.ServeCodec(NewServerCodec(conn))
}
