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

package whisperv6

import (
	"encoding/json"
	"testing"
)

var topicStringTests = []struct {
	topic TopicType
	str   string
}{
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, str: "0x00000000"},
	{topic: TopicType{0x00, 0x7f, 0x80, 0xff}, str: "0x007f80ff"},
	{topic: TopicType{0xff, 0x80, 0x7f, 0x00}, str: "0xff807f00"},
	{topic: TopicType{0xf2, 0x6e, 0x77, 0x79}, str: "0xf26e7779"},
}

func TestTopicString(t *testing.T) {
	for i, wat := range topicStringTests {
		s := wat.topic.String()
		if s != wat.str {
			t.Fatalf("failed test %d: have %s, want %s.", i, s, wat.str)
		}
	}
}

var bytesToTopicTests = []struct {
	data  []byte
	topic TopicType
}{
	{topic: TopicType{0x8f, 0x9a, 0x2b, 0x7d}, data: []byte{0x8f, 0x9a, 0x2b, 0x7d}},
	{topic: TopicType{0x00, 0x7f, 0x80, 0xff}, data: []byte{0x00, 0x7f, 0x80, 0xff}},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte{0x00, 0x00, 0x00, 0x00}},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte{0x00, 0x00, 0x00}},
	{topic: TopicType{0x01, 0x00, 0x00, 0x00}, data: []byte{0x01}},
	{topic: TopicType{0x00, 0xfe, 0x00, 0x00}, data: []byte{0x00, 0xfe}},
	{topic: TopicType{0xea, 0x1d, 0x43, 0x00}, data: []byte{0xea, 0x1d, 0x43}},
	{topic: TopicType{0x6f, 0x3c, 0xb0, 0xdd}, data: []byte{0x6f, 0x3c, 0xb0, 0xdd, 0x0f, 0x00, 0x90}},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte{}},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: nil},
}

var unmarshalTestsGood = []struct {
	topic TopicType
	data  []byte
}{
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"0x00000000"`)},
	{topic: TopicType{0x00, 0x7f, 0x80, 0xff}, data: []byte(`"0x007f80ff"`)},
	{topic: TopicType{0xff, 0x80, 0x7f, 0x00}, data: []byte(`"0xff807f00"`)},
	{topic: TopicType{0xf2, 0x6e, 0x77, 0x79}, data: []byte(`"0xf26e7779"`)},
}

var unmarshalTestsBad = []struct {
	topic TopicType
	data  []byte
}{
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"0x000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"0x0000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"0x000000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"0x0000000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"0000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"000000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"0000000000"`)},
	{topic: TopicType{0x00, 0x00, 0x00, 0x00}, data: []byte(`"abcdefg0"`)},
}

var unmarshalTestsUgly = []struct {
	topic TopicType
	data  []byte
}{
	{topic: TopicType{0x01, 0x00, 0x00, 0x00}, data: []byte(`"0x00000001"`)},
}

func TestBytesToTopic(t *testing.T) {
	for i, wat := range bytesToTopicTests {
		top := BytesToTopic(wat.data)
		if top != wat.topic {
			t.Fatalf("failed test %d: have %v, want %v.", i, t, wat.topic)
		}
	}
}

func TestUnmarshalTestsGood(t *testing.T) {
	for i, wat := range unmarshalTestsGood {
		var top TopicType
		err := json.Unmarshal(wat.data, &top)
		if err != nil {
			t.Errorf("failed test %d. input: %v. err: %v", i, wat.data, err)
		} else if top != wat.topic {
			t.Errorf("failed test %d: have %v, want %v.", i, t, wat.topic)
		}
	}
}

func TestUnmarshalTestsBad(t *testing.T) {
	// in this test UnmarshalJSON() is supposed to fail
	for i, wat := range unmarshalTestsBad {
		var top TopicType
		err := json.Unmarshal(wat.data, &top)
		if err == nil {
			t.Fatalf("failed test %d. input: %v.", i, wat.data)
		}
	}
}

func TestUnmarshalTestsUgly(t *testing.T) {
	// in this test UnmarshalJSON() is NOT supposed to fail, but result should be wrong
	for i, wat := range unmarshalTestsUgly {
		var top TopicType
		err := json.Unmarshal(wat.data, &top)
		if err != nil {
			t.Errorf("failed test %d. input: %v.", i, wat.data)
		} else if top == wat.topic {
			t.Errorf("failed test %d: have %v, want %v.", i, top, wat.topic)
		}
	}
}
