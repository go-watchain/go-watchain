// Copyright 2016 The go-ethereum Authors
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

package storage

import (
	"testing"
)

func testMemStore(l int64, branches int64, t *testing.T) {
	m := NewMemStore(nil, defaultCacheCapacity)
	teswatore(m, l, branches, t)
}

func TestMemStore128_10000(t *testing.T) {
	testMemStore(10000, 128, t)
}

func TestMemStore128_1000(t *testing.T) {
	testMemStore(1000, 128, t)
}

func TestMemStore128_100(t *testing.T) {
	testMemStore(100, 128, t)
}

func TestMemStore2_100(t *testing.T) {
	testMemStore(100, 2, t)
}

func TestMemStoreNotFound(t *testing.T) {
	m := NewMemStore(nil, defaultCacheCapacity)
	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}
