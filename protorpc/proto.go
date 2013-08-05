// Copyright 2013 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protorpc

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/binary"
	"io"
)

type ProtoReader interface {
	io.ByteReader
	io.Reader
}

// ReadProto reads a uvarint size and then a protobuf from r.
// If the size read is zero, nothing more is read.
func ReadProto(r ProtoReader, pb proto.Message) error {
	size, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	if size != 0 {
		buf := make([]byte, size)
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if pb != nil {
			return proto.Unmarshal(buf, pb)
		}
	}
	return nil
}

// WriteProto writes a uvarint size and then a protobuf to w.
// If the data takes no space (like rpc.InvalidRequest),
// only a zero size is written.
func WriteProto(w io.Writer, pb proto.Message) error {
	// Allocate enough space for the biggest uvarint
	var size [binary.MaxVarintLen64]byte

	if pb == nil {
		n := binary.PutUvarint(size[:], uint64(0))
		if _, err := w.Write(size[:n]); err != nil {
			return err
		}
		return nil
	}

	// Marshal the protobuf
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	// Write the size and data
	n := binary.PutUvarint(size[:], uint64(len(data)))
	if _, err = w.Write(size[:n]); err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	return nil
}
