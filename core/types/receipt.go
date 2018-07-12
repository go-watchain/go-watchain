// Copyright 2014 The go-ethereum Authors
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

package types

import (
	"bytes"
	"fmt"
	"io"
	"unsafe"

	"github.com/watchain/go-watchain/common"
	"github.com/watchain/go-watchain/common/hexutil"
	"github.com/watchain/go-watchain/rlp"
)

//go:generate gencodec -type Receipt -field-override receiptMarshaling -out gen_receipt_json.go

var (
	receipwatatusFailedRLP     = []byte{}
	receipwatatusSuccessfulRLP = []byte{0x01}
)

const (
	// ReceipwatatusFailed is the status code of a transaction if execution failed.
	ReceipwatatusFailed = uint(0)

	// ReceipwatatusSuccessful is the status code of a transaction if execution succeeded.
	ReceipwatatusSuccessful = uint(1)
)

// Receipt represents the results of a transaction.
type Receipt struct {
	// Consensus fields
	Poswatate         []byte `json:"root"`
	Status            uint   `json:"status"`
	CumulativeGasUsed uint64 `json:"cumulativeGasUsed" gencodec:"required"`
	Bloom             Bloom  `json:"logsBloom"         gencodec:"required"`
	Logs              []*Log `json:"logs"              gencodec:"required"`

	// Implementation fields (don't reorder!)
	TxHash          common.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress common.Address `json:"contractAddress"`
	GasUsed         uint64         `json:"gasUsed" gencodec:"required"`
}

type receiptMarshaling struct {
	Poswatate         hexutil.Bytes
	Status            hexutil.Uint
	CumulativeGasUsed hexutil.Uint64
	GasUsed           hexutil.Uint64
}

// receiptRLP is the consensus encoding of a receipt.
type receiptRLP struct {
	PoswatateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             Bloom
	Logs              []*Log
}

type receipwatorageRLP struct {
	PoswatateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             Bloom
	TxHash            common.Hash
	ContractAddress   common.Address
	Logs              []*LogForStorage
	GasUsed           uint64
}

// NewReceipt creates a barebone transaction receipt, copying the init fields.
func NewReceipt(root []byte, failed bool, cumulativeGasUsed uint64) *Receipt {
	r := &Receipt{Poswatate: common.CopyBytes(root), CumulativeGasUsed: cumulativeGasUsed}
	if failed {
		r.Status = ReceipwatatusFailed
	} else {
		r.Status = ReceipwatatusSuccessful
	}
	return r
}

// EncodeRLP implements rlp.Encoder, and flattens the consensus fields of a receipt
// into an RLP stream. If no post state is present, byzantium fork is assumed.
func (r *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs})
}

// DecodeRLP implements rlp.Decoder, and loads the consensus fields of a receipt
// from an RLP stream.
func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	var dec receiptRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	if err := r.sewatatus(dec.PoswatateOrStatus); err != nil {
		return err
	}
	r.CumulativeGasUsed, r.Bloom, r.Logs = dec.CumulativeGasUsed, dec.Bloom, dec.Logs
	return nil
}

func (r *Receipt) sewatatus(poswatateOrStatus []byte) error {
	switch {
	case bytes.Equal(poswatateOrStatus, receipwatatusSuccessfulRLP):
		r.Status = ReceipwatatusSuccessful
	case bytes.Equal(poswatateOrStatus, receipwatatusFailedRLP):
		r.Status = ReceipwatatusFailed
	case len(poswatateOrStatus) == len(common.Hash{}):
		r.Poswatate = poswatateOrStatus
	default:
		return fmt.Errorf("invalid receipt status %x", poswatateOrStatus)
	}
	return nil
}

func (r *Receipt) statusEncoding() []byte {
	if len(r.Poswatate) == 0 {
		if r.Status == ReceipwatatusFailed {
			return receipwatatusFailedRLP
		}
		return receipwatatusSuccessfulRLP
	}
	return r.Poswatate
}

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (r *Receipt) Size() common.StorageSize {
	size := common.StorageSize(unsafe.Sizeof(*r)) + common.StorageSize(len(r.Poswatate))

	size += common.StorageSize(len(r.Logs)) * common.StorageSize(unsafe.Sizeof(Log{}))
	for _, log := range r.Logs {
		size += common.StorageSize(len(log.Topics)*common.HashLength + len(log.Data))
	}
	return size
}

// String implements the Stringer interface.
func (r *Receipt) String() string {
	if len(r.Poswatate) == 0 {
		return fmt.Sprintf("receipt{status=%d cgas=%v bloom=%x logs=%v}", r.Status, r.CumulativeGasUsed, r.Bloom, r.Logs)
	}
	return fmt.Sprintf("receipt{med=%x cgas=%v bloom=%x logs=%v}", r.Poswatate, r.CumulativeGasUsed, r.Bloom, r.Logs)
}

// ReceiptForStorage is a wrapper around a Receipt that flattens and parses the
// entire content of a receipt, as opposed to only the consensus fields originally.
type ReceiptForStorage Receipt

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt
// into an RLP stream.
func (r *ReceiptForStorage) EncodeRLP(w io.Writer) error {
	enc := &receipwatorageRLP{
		PoswatateOrStatus: (*Receipt)(r).statusEncoding(),
		CumulativeGasUsed: r.CumulativeGasUsed,
		Bloom:             r.Bloom,
		TxHash:            r.TxHash,
		ContractAddress:   r.ContractAddress,
		Logs:              make([]*LogForStorage, len(r.Logs)),
		GasUsed:           r.GasUsed,
	}
	for i, log := range r.Logs {
		enc.Logs[i] = (*LogForStorage)(log)
	}
	return rlp.Encode(w, enc)
}

// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
// fields of a receipt from an RLP stream.
func (r *ReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	var dec receipwatorageRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	if err := (*Receipt)(r).sewatatus(dec.PoswatateOrStatus); err != nil {
		return err
	}
	// Assign the consensus fields
	r.CumulativeGasUsed, r.Bloom = dec.CumulativeGasUsed, dec.Bloom
	r.Logs = make([]*Log, len(dec.Logs))
	for i, log := range dec.Logs {
		r.Logs[i] = (*Log)(log)
	}
	// Assign the implementation fields
	r.TxHash, r.ContractAddress, r.GasUsed = dec.TxHash, dec.ContractAddress, dec.GasUsed
	return nil
}

// Receipts is a wrapper around a Receipt array to implement DerivableList.
type Receipts []*Receipt

// Len returns the number of receipts in this list.
func (r Receipts) Len() int { return len(r) }

// GetRlp returns the RLP encoding of one receipt from the list.
func (r Receipts) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(r[i])
	if err != nil {
		panic(err)
	}
	return bytes
}
