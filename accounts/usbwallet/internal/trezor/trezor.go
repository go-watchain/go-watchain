// Copyright 2017 The go-ethereum Authors
// This file is part of the go-watereum library.
//
// The go-watereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-watereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-watereum library. If not, see <http://www.gnu.org/licenses/>.

// This file contains the implementation for interacting with the Trezor hardware
// wallets. The wire protocol spec can be found on the SatoshiLabs website:
// https://doc.satoshilabs.com/trezor-tech/api-protobuf.html

//go:generate protoc --go_out=import_path=trezor:. types.proto messages.proto

// Package trezor contains the wire protocol wrapper in Go.
package trezor

import (
	"reflect"

	"github.com/golang/protobuf/proto"
)

// Type returns the protocol buffer type number of a specific message. If the
// message is nil, this method panics!
func Type(msg proto.Message) uint16 {
	return uint16(MessageType_value["MessageType_"+reflect.TypeOf(msg).Elem().Name()])
}

// Name returns the friendly message type name of a specific protocol buffer
// type numbers.
func Name(kind uint16) string {
	name := MessageType_name[int32(kind)]
	if len(name) < 12 {
		return name
	}
	return name[12:]
}
